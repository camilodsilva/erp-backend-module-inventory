package product

import "fmt"

type UpdateUseCase struct {
	repository Repository
}

func NewUpdateUseCase(repository Repository) *UpdateUseCase {
	return &UpdateUseCase{repository: repository}
}

func (u *UpdateUseCase) Execute(tenantID, id, actorID string, draft Draft) (Product, error) {
	current, err := u.repository.FindByID(tenantID, id)
	if err != nil {
		return Product{}, fmt.Errorf("error trying to find product before update: %w", err)
	}

	updated := current.Update(draft, actorID)

	saved, err := u.repository.Update(tenantID, updated)
	if err != nil {
		return Product{}, fmt.Errorf("error trying to update product: %w", err)
	}

	return saved, nil
}
