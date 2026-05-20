package product

type ProductFactory struct {
	idGenerator IDGenerator
}

func NewProductFactory(idGenerator IDGenerator) *ProductFactory {
	return &ProductFactory{idGenerator: idGenerator}
}

func (f *ProductFactory) Create(actorID string, draft Draft) Product {
	return Product{
		ID:                      f.idGenerator.Generate(),
		Title:                   draft.Title,
		Description:             draft.Description,
		SKU:                     draft.SKU,
		EAN:                     draft.EAN,
		Unit:                    draft.Unit,
		UnitPrice:               draft.UnitPrice,
		StockQuantity:           draft.StockQuantity,
		IsActive:                true,
		FiscalProfileExternalID: draft.FiscalProfileExternalID,
		CreatedBy:               actorID,
		UpdatedBy:               actorID,
	}
}
