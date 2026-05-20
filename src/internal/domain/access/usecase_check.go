package access

type CheckUseCase struct{}

func NewCheckUseCase() *CheckUseCase {
	return &CheckUseCase{}
}

func (u *CheckUseCase) Execute(draft Draft) (AccessStatus, error) {
	return NewAccessStatus(draft.CanRead, draft.CanWrite), nil
}
