package product

import "fmt"

type FindByIDUseCase struct {
	repository Repository
}

func NewFindByIDUseCase(repository Repository) *FindByIDUseCase {
	return &FindByIDUseCase{repository: repository}
}

func (u *FindByIDUseCase) Execute(tenantID, id string) (Product, error) {
	p, err := u.repository.FindByID(tenantID, id)
	if err != nil {
		return Product{}, fmt.Errorf("error trying to find product: %w", err)
	}

	return p, nil
}
