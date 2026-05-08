// backend/internal/application/usecase/list/constructors.go
package list

func NewListUsecase(
	listReader ListReader,
	listCreator ListCreator,
	listPatcher ListPatcher,
	imageReader ListImageReader,
	imageByIDReader ListImageByIDReader,
) *ListUsecase {
	uc := &ListUsecase{
		listReader:  listReader,
		listLister:  nil, // auto-wire below
		listCreator: listCreator,
		listUpdater: nil, // auto-wire below
		listPatcher: listPatcher,

		// Firebase Storage 移行後:
		// - imageReader / imageByIDReader は Firestore の
		//   /lists/{listId}/images/{imageId} record を読むための port
		// - backend は GCS signed URL / GCS object / bucket を扱わない
		imageReader:     imageReader,
		imageByIDReader: imageByIDReader,

		// Firestore subcollection repository.
		// imageReader または imageByIDReader が ListImageRecordRepository を満たす場合に auto-wire する。
		listImageRecordRepo: nil,

		// list 本体の primary imageId を更新するための port。
		listPrimaryImageSetter: nil,
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
	// listCreator -> ListLister / ListUpdater
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
	// listPatcher -> PrimarySetter
	// ---------------------------------------------------------
	if uc.listPrimaryImageSetter == nil && listPatcher != nil {
		if setter, ok := any(listPatcher).(ListPrimaryImageSetter); ok {
			uc.listPrimaryImageSetter = setter
		}
	}

	// ---------------------------------------------------------
	// Firestore list image record repo auto-wire
	//
	// Source of truth:
	// - /lists/{listId}/images/{imageId}
	//
	// Priority:
	// 1. imageReader
	// 2. imageByIDReader
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

	return uc
}
