package product

import "fmt"

type CreateUseCase struct {
	repository Repository
	factory    *ProductFactory
}

func NewCreateUseCase(repository Repository, factory *ProductFactory) *CreateUseCase {
	return &CreateUseCase{repository: repository, factory: factory}
}

func (u *CreateUseCase) Execute(tenantID, actorID string, draft Draft) (Product, error) {
	p := u.factory.Create(actorID, draft)

	created, err := u.repository.Create(tenantID, p)
	if err != nil {
		return Product{}, fmt.Errorf("error trying to create product: %w", err)
	}

	return created, nil
}
