package product

import "fmt"

type FindAllUseCase struct {
	repository Repository
}

func NewFindAllUseCase(repository Repository) *FindAllUseCase {
	return &FindAllUseCase{repository: repository}
}

func (u *FindAllUseCase) Execute(tenantID string, page, size int, q string) (Page, error) {
	result, err := u.repository.FindAll(tenantID, page, size, q)
	if err != nil {
		return Page{}, fmt.Errorf("error trying to list products: %w", err)
	}

	return result, nil
}
