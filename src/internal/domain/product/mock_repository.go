package product

type MockProductRepository struct {
	CreateFn      func(tenantID string, p Product) (Product, error)
	FindAllFn     func(tenantID string, page, size int) (Page, error)
	FindByIDFn    func(tenantID, id string) (Product, error)
	UpdateFn      func(tenantID string, p Product) (Product, error)
	SoftDeleteFn  func(tenantID, id, deletedBy string) error
}

func (m *MockProductRepository) Create(tenantID string, p Product) (Product, error) {
	return m.CreateFn(tenantID, p)
}

func (m *MockProductRepository) FindAll(tenantID string, page, size int) (Page, error) {
	return m.FindAllFn(tenantID, page, size)
}

func (m *MockProductRepository) FindByID(tenantID, id string) (Product, error) {
	return m.FindByIDFn(tenantID, id)
}

func (m *MockProductRepository) Update(tenantID string, p Product) (Product, error) {
	return m.UpdateFn(tenantID, p)
}

func (m *MockProductRepository) SoftDelete(tenantID, id, deletedBy string) error {
	return m.SoftDeleteFn(tenantID, id, deletedBy)
}
