package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"pymego/internal/domain"
)

// ---------------------------------------------------------------------------
// Fakes de repositorios y servicios para tests unitarios.
// ---------------------------------------------------------------------------

type fakeUserRepo struct {
	usersByEmail map[string]*domain.User
	usersByID    map[string]*domain.User
	createErr    error
	seq          int
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		usersByEmail: map[string]*domain.User{},
		usersByID:    map[string]*domain.User{},
	}
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if _, ok := f.usersByEmail[u.Email]; ok {
		return fmt.Errorf("%w: email duplicado", domain.ErrConflict)
	}
	f.seq++
	u.ID = fmt.Sprintf("user-%d", f.seq)
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now, now
	f.usersByEmail[u.Email] = u
	f.usersByID[u.ID] = u
	return nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := f.usersByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.usersByEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, ok := f.usersByEmail[email]
	return ok, nil
}

type fakeTokenIssuer struct {
	token     string
	expiresAt time.Time
	err       error
	gotUserID string
	gotRole   string
	calls     int
}

func (f *fakeTokenIssuer) Generate(userID, role string) (string, time.Time, error) {
	f.calls++
	f.gotUserID = userID
	f.gotRole = role
	if f.err != nil {
		return "", time.Time{}, f.err
	}
	return f.token, f.expiresAt, nil
}

type fakeCategoryRepo struct {
	items     map[string]*domain.Category
	seq       int
	createErr error
	getErr    error
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{items: map[string]*domain.Category{}}
}

func (f *fakeCategoryRepo) Create(_ context.Context, c *domain.Category) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.seq++
	c.ID = fmt.Sprintf("cat-%d", f.seq)
	now := time.Now()
	c.CreatedAt, c.UpdatedAt = now, now
	f.items[c.ID] = c
	return nil
}

func (f *fakeCategoryRepo) GetByID(_ context.Context, id string) (*domain.Category, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (f *fakeCategoryRepo) List(_ context.Context) ([]domain.Category, error) {
	out := make([]domain.Category, 0, len(f.items))
	for _, c := range f.items {
		out = append(out, *c)
	}
	return out, nil
}

func (f *fakeCategoryRepo) Update(_ context.Context, c *domain.Category) error {
	if _, ok := f.items[c.ID]; !ok {
		return domain.ErrNotFound
	}
	f.items[c.ID] = c
	return nil
}

func (f *fakeCategoryRepo) Delete(_ context.Context, id string) error {
	if _, ok := f.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.items, id)
	return nil
}

type fakeProductRepo struct {
	products         map[string]*domain.Product
	variants         map[string]*domain.ProductVariant
	seq              int
	createErr        error
	createVariantErr error
	updateVariantErr error
	adjustErr        error
}

func newFakeProductRepo() *fakeProductRepo {
	return &fakeProductRepo{
		products: map[string]*domain.Product{},
		variants: map[string]*domain.ProductVariant{},
	}
}

func (f *fakeProductRepo) Create(_ context.Context, p *domain.Product) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.seq++
	p.ID = fmt.Sprintf("prod-%d", f.seq)
	now := time.Now()
	p.CreatedAt, p.UpdatedAt = now, now
	f.products[p.ID] = p
	return nil
}

func (f *fakeProductRepo) GetByID(_ context.Context, id string) (*domain.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (f *fakeProductRepo) List(_ context.Context) ([]domain.Product, error) {
	out := make([]domain.Product, 0, len(f.products))
	for _, p := range f.products {
		out = append(out, *p)
	}
	return out, nil
}

func (f *fakeProductRepo) Update(_ context.Context, p *domain.Product) error {
	if _, ok := f.products[p.ID]; !ok {
		return domain.ErrNotFound
	}
	f.products[p.ID] = p
	return nil
}

func (f *fakeProductRepo) Delete(_ context.Context, id string) error {
	if _, ok := f.products[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.products, id)
	return nil
}

func (f *fakeProductRepo) CreateVariant(_ context.Context, v *domain.ProductVariant) error {
	if f.createVariantErr != nil {
		return f.createVariantErr
	}
	for _, existing := range f.variants {
		if existing.SKU == v.SKU {
			return fmt.Errorf("%w: SKU duplicado", domain.ErrConflict)
		}
	}
	f.seq++
	v.ID = fmt.Sprintf("var-%d", f.seq)
	now := time.Now()
	v.CreatedAt, v.UpdatedAt = now, now
	f.variants[v.ID] = v
	return nil
}

func (f *fakeProductRepo) GetVariantByID(_ context.Context, id string) (*domain.ProductVariant, error) {
	v, ok := f.variants[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copyVariant := *v
	return &copyVariant, nil
}

func (f *fakeProductRepo) ListVariantsByProduct(_ context.Context, productID string) ([]domain.ProductVariant, error) {
	out := make([]domain.ProductVariant, 0)
	for _, v := range f.variants {
		if v.ProductID == productID {
			out = append(out, *v)
		}
	}
	return out, nil
}

func (f *fakeProductRepo) UpdateVariant(_ context.Context, v *domain.ProductVariant) error {
	if f.updateVariantErr != nil {
		return f.updateVariantErr
	}
	if _, ok := f.variants[v.ID]; !ok {
		return domain.ErrNotFound
	}
	f.variants[v.ID] = v
	return nil
}

func (f *fakeProductRepo) DeleteVariant(_ context.Context, id string) error {
	if _, ok := f.variants[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.variants, id)
	return nil
}

func (f *fakeProductRepo) AdjustStockQuantity(_ context.Context, id string, delta int) (*domain.ProductVariant, error) {
	if f.adjustErr != nil {
		return nil, f.adjustErr
	}
	v, ok := f.variants[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if v.StockQuantity+delta < 0 {
		return nil, domain.NewValidationError("stock", "stock insuficiente")
	}
	v.StockQuantity += delta
	copyVariant := *v
	return &copyVariant, nil
}

type fakeOrderRepo struct {
	orders    map[string]*domain.OrderLog
	seq       int
	createErr error
	updateErr error
}

func newFakeOrderRepo() *fakeOrderRepo {
	return &fakeOrderRepo{orders: map[string]*domain.OrderLog{}}
}

func (f *fakeOrderRepo) Create(_ context.Context, o *domain.OrderLog) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.seq++
	o.ID = fmt.Sprintf("order-%d", f.seq)
	now := time.Now()
	o.CreatedAt, o.UpdatedAt = now, now
	f.orders[o.ID] = o
	return nil
}

func (f *fakeOrderRepo) GetByID(_ context.Context, id string) (*domain.OrderLog, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return o, nil
}

func (f *fakeOrderRepo) List(_ context.Context) ([]domain.OrderLog, error) {
	out := make([]domain.OrderLog, 0, len(f.orders))
	for _, o := range f.orders {
		out = append(out, *o)
	}
	return out, nil
}

func (f *fakeOrderRepo) UpdateStatus(_ context.Context, id string, status domain.OrderStatus) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	o, ok := f.orders[id]
	if !ok {
		return domain.ErrNotFound
	}
	o.Status = status
	return nil
}

type fakeTx struct {
	calls int
	err   error
}

func (f *fakeTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	return fn(ctx)
}

// ListActive devuelve solo las categorías activas (vitrina pública).
func (f *fakeCategoryRepo) ListActive(_ context.Context) ([]domain.Category, error) {
	out := make([]domain.Category, 0, len(f.items))
	for _, c := range f.items {
		if c.IsActive {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// hasSellableVariant indica si el producto tiene al menos una variante activa
// con stock disponible.
func (f *fakeProductRepo) hasSellableVariant(productID string) bool {
	for _, v := range f.variants {
		if v.ProductID == productID && v.IsActive && v.StockQuantity > 0 {
			return true
		}
	}
	return false
}

// ListSellable devuelve productos activos con stock disponible, opcionalmente
// filtrados por categoría.
func (f *fakeProductRepo) ListSellable(_ context.Context, categoryID *string) ([]domain.Product, error) {
	out := make([]domain.Product, 0)
	for _, p := range f.products {
		if !p.IsActive {
			continue
		}
		if categoryID != nil && (p.CategoryID == nil || *p.CategoryID != *categoryID) {
			continue
		}
		if !f.hasSellableVariant(p.ID) {
			continue
		}
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// GetSellableByID devuelve un producto vendible o ErrNotFound.
func (f *fakeProductRepo) GetSellableByID(_ context.Context, id string) (*domain.Product, error) {
	p, ok := f.products[id]
	if !ok || !p.IsActive || !f.hasSellableVariant(id) {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

// ListActiveVariantsByProduct devuelve las variantes activas de un producto.
func (f *fakeProductRepo) ListActiveVariantsByProduct(_ context.Context, productID string) ([]domain.ProductVariant, error) {
	out := make([]domain.ProductVariant, 0)
	for _, v := range f.variants {
		if v.ProductID == productID && v.IsActive {
			out = append(out, *v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
