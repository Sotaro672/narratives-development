// backend/internal/application/usecase/list/constructors.go
package list

func NewListUsecase(
	listReader ListReader,
	listCreator ListCreator, // ✅ 追加：唯一の入口で creator も受け取る
	listPatcher ListPatcher,
	imageReader ListImageReader,
	imageByIDReader ListImageByIDReader,
	imageObjectSaver ListImageObjectSaver,
) *ListUsecase {
	uc := &ListUsecase{
		listReader:  listReader,
		listLister:  nil, // auto-wire below
		listCreator: listCreator,
		listUpdater: nil, // auto-wire below
		listPatcher: listPatcher,

		imageReader:      imageReader,
		imageByIDReader:  imageByIDReader,
		imageObjectSaver: imageObjectSaver,

		imageSignedURLIssuer: nil, // auto-wire below

		listImageRecordRepo:    nil, // auto-wire below
		listPrimaryImageSetter: nil, // auto-wire below

		listImageDeleter: nil, // auto-wire below
	}

	// ---------------------------------------------------------
	// listReader -> ListLister / ListUpdater / PrimarySetter
	// ---------------------------------------------------------
	if listReader != nil {
		if lister, ok := any(listReader).(ListLister); ok {
			uc.listLister = lister
		}
		if updater, ok := any(listReader).(ListUpdater); ok {
			uc.listUpdater = updater
		}
		if setter, ok := any(listReader).(ListPrimaryImageSetter); ok {
			uc.listPrimaryImageSetter = setter
		}
	}

	// ---------------------------------------------------------
	// listCreator -> ListLister / ListUpdater (保険)
	// ---------------------------------------------------------
	if uc.listLister == nil && listCreator != nil {
		if lister, ok := any(listCreator).(ListLister); ok {
			uc.listLister = lister
		}
	}
	if uc.listUpdater == nil && listCreator != nil {
		if updater, ok := any(listCreator).(ListUpdater); ok {
			uc.listUpdater = updater
		}
	}

	// ---------------------------------------------------------
	// listPatcher -> PrimarySetter (保険)
	// ---------------------------------------------------------
	if uc.listPrimaryImageSetter == nil && listPatcher != nil {
		if setter, ok := any(listPatcher).(ListPrimaryImageSetter); ok {
			uc.listPrimaryImageSetter = setter
		}
	}

	// ---------------------------------------------------------
	// imageObjectSaver -> SignedURLIssuer
	// ---------------------------------------------------------
	if imageObjectSaver != nil {
		if issuer, ok := any(imageObjectSaver).(ListImageSignedURLIssuer); ok {
			uc.imageSignedURLIssuer = issuer
		}
	}

	// ---------------------------------------------------------
	// Firestore record repo auto-wire
	// 優先順位: imageReader -> imageByIDReader -> imageObjectSaver
	// ---------------------------------------------------------
	if imageReader != nil {
		if repo, ok := any(imageReader).(ListImageRecordRepository); ok {
			uc.listImageRecordRepo = repo
		}
	}
	if uc.listImageRecordRepo == nil && imageByIDReader != nil {
		if repo, ok := any(imageByIDReader).(ListImageRecordRepository); ok {
			uc.listImageRecordRepo = repo
		}
	}
	if uc.listImageRecordRepo == nil && imageObjectSaver != nil {
		if repo, ok := any(imageObjectSaver).(ListImageRecordRepository); ok {
			uc.listImageRecordRepo = repo
		}
	}

	// ---------------------------------------------------------
	// Delete auto-wire
	// 優先順位: imageReader -> imageByIDReader -> imageObjectSaver
	// ---------------------------------------------------------
	if uc.listImageDeleter == nil && imageReader != nil {
		if d, ok := any(imageReader).(ListImageDeleter); ok {
			uc.listImageDeleter = d
		}
	}
	if uc.listImageDeleter == nil && imageByIDReader != nil {
		if d, ok := any(imageByIDReader).(ListImageDeleter); ok {
			uc.listImageDeleter = d
		}
	}
	if uc.listImageDeleter == nil && imageObjectSaver != nil {
		if d, ok := any(imageObjectSaver).(ListImageDeleter); ok {
			uc.listImageDeleter = d
		}
	}

	return uc
}
