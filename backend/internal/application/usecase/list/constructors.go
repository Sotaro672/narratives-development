// backend/internal/application/usecase/list/constructors.go
//
// Responsibility:
// - ListUsecase の「依存注入と自動配線」を担当する（DI/auto-wire）。
// - 実装側(repo/adapter)が追加インターフェースを満たす場合に配線する。
//
// Features:
// - NewListUsecase / NewListUsecaseWithCreator
// - listLister/listUpdater/signedUrlIssuer の auto-wire
package list

func NewListUsecase(
	listReader ListReader,
	listPatcher ListPatcher,
	imageReader ListImageReader,
	imageByIDReader ListImageByIDReader,
	imageObjectSaver ListImageObjectSaver,
) *ListUsecase {
	uc := &ListUsecase{
		listReader:       listReader,
		listLister:       nil, // auto-wire below
		listCreator:      nil,
		listUpdater:      nil, // auto-wire below
		listPatcher:      listPatcher,
		imageReader:      imageReader,
		imageByIDReader:  imageByIDReader,
		imageObjectSaver: imageObjectSaver,

		imageSignedURLIssuer: nil, // auto-wire below
	}

	// listReader(実体はrepo)が ListLister/ListUpdater を実装していれば自動で配線
	if listReader != nil {
		if lister, ok := any(listReader).(ListLister); ok {
			uc.listLister = lister
		}
		if updater, ok := any(listReader).(ListUpdater); ok {
			uc.listUpdater = updater
		}
	}

	// signed-url: imageObjectSaver が issuer を実装していれば自動で配線
	if imageObjectSaver != nil {
		if issuer, ok := any(imageObjectSaver).(ListImageSignedURLIssuer); ok {
			uc.imageSignedURLIssuer = issuer
		}
	}

	return uc
}

func NewListUsecaseWithCreator(
	listReader ListReader,
	listCreator ListCreator,
	listPatcher ListPatcher,
	imageReader ListImageReader,
	imageByIDReader ListImageByIDReader,
	imageObjectSaver ListImageObjectSaver,
) *ListUsecase {
	uc := &ListUsecase{
		listReader:       listReader,
		listLister:       nil, // auto-wire below
		listCreator:      listCreator,
		listUpdater:      nil, // auto-wire below
		listPatcher:      listPatcher,
		imageReader:      imageReader,
		imageByIDReader:  imageByIDReader,
		imageObjectSaver: imageObjectSaver,

		imageSignedURLIssuer: nil, // auto-wire below
	}

	// listReader が ListLister/ListUpdater を実装していれば優先
	if listReader != nil {
		if lister, ok := any(listReader).(ListLister); ok {
			uc.listLister = lister
		}
		if updater, ok := any(listReader).(ListUpdater); ok {
			uc.listUpdater = updater
		}
	}
	// 念のため: listCreator(同じrepoを渡しているケース)が実装していれば配線
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

	// signed-url: imageObjectSaver が issuer を実装していれば自動で配線
	if imageObjectSaver != nil {
		if issuer, ok := any(imageObjectSaver).(ListImageSignedURLIssuer); ok {
			uc.imageSignedURLIssuer = issuer
		}
	}

	return uc
}
