package product

import "fmt"

type DeleteUseCase struct {
	repository Repository
}

func NewDeleteUseCase(repository Repository) *DeleteUseCase {
	return &DeleteUseCase{repository: repository}
}

func (u *DeleteUseCase) Execute(tenantID, id, actorID string) error {
	if err := u.repository.SoftDelete(tenantID, id, actorID); err != nil {
		return fmt.Errorf("error trying to delete product: %w", err)
	}

	return nil
}
