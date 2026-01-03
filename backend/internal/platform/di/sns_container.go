// backend/internal/platform/di/sns_container.go
package di

import (
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"cloud.google.com/go/firestore"
	firebaseauth "firebase.google.com/go/v4/auth"

	"narratives/internal/adapters/in/http/middleware"
	snshttp "narratives/internal/adapters/in/http/sns"
	snshandler "narratives/internal/adapters/in/http/sns/handler"
	outfs "narratives/internal/adapters/out/firestore"
	snsquery "narratives/internal/application/query/sns"
	appresolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"
	ldom "narratives/internal/domain/list"
)

// SNSDeps is a buyer-facing (sns) HTTP dependency set.
type SNSDeps struct {
	// Handlers
	List             http.Handler
	Inventory        http.Handler
	ProductBlueprint http.Handler
	Model            http.Handler
	Catalog          http.Handler

	TokenBlueprint http.Handler // patch

	// ✅ NEW: name resolver endpoints
	Company http.Handler
	Brand   http.Handler

	// ✅ auth entry (cart empty ok)
	SignIn http.Handler

	// ✅ auth onboarding resources
	User            http.Handler
	ShippingAddress http.Handler
	BillingAddress  http.Handler
	Avatar          http.Handler

	// ✅ NEW: avatar state
	AvatarState http.Handler

	// ✅ NEW: wallet
	Wallet http.Handler

	// ✅ NEW: cart
	Cart http.Handler

	// ✅ NEW: preview
	Preview http.Handler

	// ✅ NEW: posts
	Post http.Handler

	// ✅ NEW: payment (order context / checkout)
	Payment http.Handler
}

// NewSNSDeps wires SNS handlers.
func NewSNSDeps(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,
	catalogQ *snsquery.SNSCatalogQuery,
) SNSDeps {
	return NewSNSDepsWithNameResolver(
		listUC,
		invUC,
		pbUC,
		modelUC,
		tokenBlueprintUC,
		nil, // nameResolver
		catalogQ,
	)
}

func NewSNSDepsWithNameResolver(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,
	nameResolver *appresolver.NameResolver,
	catalogQ *snsquery.SNSCatalogQuery,
) SNSDeps {
	return NewSNSDepsWithNameResolverAndOrgHandlers(
		listUC,
		invUC,
		pbUC,
		modelUC,
		tokenBlueprintUC,
		nil,
		nil,
		nameResolver,
		catalogQ,
	)
}

func NewSNSDepsWithNameResolverAndOrgHandlers(
	listUC *usecase.ListUsecase,
	invUC *usecase.InventoryUsecase,
	pbUC *usecase.ProductBlueprintUsecase,
	modelUC *usecase.ModelUsecase,
	tokenBlueprintUC *usecase.TokenBlueprintUsecase,

	companyUC *usecase.CompanyUsecase,
	brandUC *usecase.BrandUsecase,

	nameResolver *appresolver.NameResolver,
	catalogQ *snsquery.SNSCatalogQuery,
) SNSDeps {
	if catalogQ != nil && nameResolver != nil && catalogQ.NameResolver == nil {
		catalogQ.NameResolver = nameResolver
	}

	var listHandler http.Handler
	var invHandler http.Handler
	var pbHandler http.Handler
	var modelHandler http.Handler
	var catalogHandler http.Handler
	var tokenBlueprintHandler http.Handler
	var companyHandler http.Handler
	var brandHandler http.Handler

	if listUC != nil {
		listHandler = snshandler.NewSNSListHandler(listUC)
	}
	if invUC != nil {
		invHandler = snshandler.NewSNSInventoryHandler(invUC)
	}
	if pbUC != nil {
		pbHandler = snshandler.NewSNSProductBlueprintHandler(pbUC)
		if nameResolver != nil {
			setOptionalResolverField(pbHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(pbHandler, "NameResolver", nameResolver)
		}
	}
	if modelUC != nil {
		modelHandler = snshandler.NewSNSModelHandler(modelUC)
	}
	if catalogQ != nil {
		catalogHandler = snshandler.NewSNSCatalogHandler(catalogQ)
	}
	if companyUC != nil {
		companyHandler = snshandler.NewSNSCompanyHandler(companyUC)
	}
	if brandUC != nil {
		brandHandler = snshandler.NewSNSBrandHandler(brandUC)
	}

	if tokenBlueprintUC != nil {
		tokenBlueprintHandler = snshandler.NewSNSTokenBlueprintHandler(tokenBlueprintUC)
		if nameResolver != nil {
			setOptionalResolverField(tokenBlueprintHandler, "BrandNameResolver", nameResolver)
			setOptionalResolverField(tokenBlueprintHandler, "CompanyNameResolver", nameResolver)
			setOptionalResolverField(tokenBlueprintHandler, "NameResolver", nameResolver)
		}
		imgResolver := appresolver.NewImageURLResolver("")
		setOptionalResolverField(tokenBlueprintHandler, "ImageResolver", imgResolver)
		setOptionalResolverField(tokenBlueprintHandler, "ImageURLResolver", imgResolver)
		setOptionalResolverField(tokenBlueprintHandler, "IconURLResolver", imgResolver)
	}

	return SNSDeps{
		List:             listHandler,
		Inventory:        invHandler,
		ProductBlueprint: pbHandler,
		Model:            modelHandler,
		Catalog:          catalogHandler,
		TokenBlueprint:   tokenBlueprintHandler,

		Company: companyHandler,
		Brand:   brandHandler,

		SignIn:          nil,
		User:            nil,
		ShippingAddress: nil,
		BillingAddress:  nil,
		Avatar:          nil,
		AvatarState:     nil,
		Wallet:          nil,
		Cart:            nil,
		Preview:         nil,
		Post:            nil,
		Payment:         nil,
	}
}

// RegisterSNSFromContainer registers SNS routes using *Container.
func RegisterSNSFromContainer(mux *http.ServeMux, cont *Container) {
	if mux == nil || cont == nil {
		return
	}

	depsAny := any(cont.RouterDeps())

	// catalog query
	var catalogQ *snsquery.SNSCatalogQuery
	{
		if x, ok := any(cont).(interface {
			SNSCatalogQuery() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.SNSCatalogQuery()
		} else if x, ok := any(cont).(interface {
			GetSNSCatalogQuery() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.GetSNSCatalogQuery()
		} else if x, ok := any(cont).(interface {
			CatalogQuery() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.CatalogQuery()
		} else if x, ok := any(cont).(interface {
			SNSCatalogQ() *snsquery.SNSCatalogQuery
		}); ok {
			catalogQ = x.SNSCatalogQ()
		}
	}

	// name resolver
	var nameResolver *appresolver.NameResolver
	{
		if x, ok := any(cont).(interface {
			SNSNameResolver() *appresolver.NameResolver
		}); ok {
			nameResolver = x.SNSNameResolver()
		} else if x, ok := any(cont).(interface {
			GetSNSNameResolver() *appresolver.NameResolver
		}); ok {
			nameResolver = x.GetSNSNameResolver()
		}
		if nameResolver == nil {
			nameResolver = getSNSNameResolverFieldBestEffort(cont)
		}
	}

	// ✅ list repo（重要）:
	// cart_query / preview_query が “正規化済み List” を使うために必須
	var listRepo ldom.Repository
	{
		// 1) Container 経由（明示 getter があるなら最優先）
		if x, ok := any(cont).(interface {
			ListRepository() ldom.Repository
		}); ok {
			listRepo = x.ListRepository()
		} else if x, ok := any(cont).(interface {
			GetListRepository() ldom.Repository
		}); ok {
			listRepo = x.GetListRepository()
		} else if x, ok := any(cont).(interface {
			ListRepo() ldom.Repository
		}); ok {
			listRepo = x.ListRepo()
		} else if x, ok := any(cont).(interface {
			GetListRepo() ldom.Repository
		}); ok {
			listRepo = x.GetListRepo()
		}

		// 2) RouterDeps フィールド（best-effort）
		if listRepo == nil {
			listRepo = getFieldPtr[ldom.Repository](depsAny, "ListRepo", "ListRepository")
		}

		// 3) ✅ 最後の砦：この場で Firestore から ListRepo を確実に構築する
		//    ※ これで「listRepo=false」の状態を潰す
		if listRepo == nil {
			if fs := getFirestoreClientBestEffort(cont, depsAny); fs != nil {
				listRepo = outfs.NewListRepositoryFS(fs)
			}
		}
	}

	// ✅ cart/preview queries
	var cartQ *snsquery.SNSCartQuery
	{
		if x, ok := any(cont).(interface {
			SNSCartQuery() *snsquery.SNSCartQuery
		}); ok {
			cartQ = x.SNSCartQuery()
		} else if x, ok := any(cont).(interface {
			GetSNSCartQuery() *snsquery.SNSCartQuery
		}); ok {
			cartQ = x.GetSNSCartQuery()
		} else if x, ok := any(cont).(interface {
			CartQuery() *snsquery.SNSCartQuery
		}); ok {
			cartQ = x.CartQuery()
		} else if x, ok := any(cont).(interface {
			SNSCartQ() *snsquery.SNSCartQuery
		}); ok {
			cartQ = x.SNSCartQ()
		}
		if cartQ == nil {
			cartQ = getSNSCartQueryFieldBestEffort(cont)
		}
		if cartQ != nil && nameResolver != nil && cartQ.Resolver == nil {
			cartQ.Resolver = nameResolver
		}
		if cartQ != nil && listRepo != nil && cartQ.ListRepo == nil {
			cartQ.ListRepo = listRepo
		}
	}

	var previewQ *snsquery.SNSPreviewQuery
	{
		if x, ok := any(cont).(interface {
			SNSPreviewQuery() *snsquery.SNSPreviewQuery
		}); ok {
			previewQ = x.SNSPreviewQuery()
		} else if x, ok := any(cont).(interface {
			GetSNSPreviewQuery() *snsquery.SNSPreviewQuery
		}); ok {
			previewQ = x.GetSNSPreviewQuery()
		} else if x, ok := any(cont).(interface {
			PreviewQuery() *snsquery.SNSPreviewQuery
		}); ok {
			previewQ = x.PreviewQuery()
		} else if x, ok := any(cont).(interface {
			SNSPreviewQ() *snsquery.SNSPreviewQuery
		}); ok {
			previewQ = x.SNSPreviewQ()
		}
		if previewQ == nil {
			previewQ = getSNSPreviewQueryFieldBestEffort(cont)
		}
		if previewQ != nil && nameResolver != nil && previewQ.Resolver == nil {
			previewQ.Resolver = nameResolver
		}
		// preview 側も listRepo を持っている場合だけ best-effort で注入（フィールドが無い実装でも安全）
		if previewQ != nil && listRepo != nil {
			setOptionalResolverField(previewQ, "ListRepo", listRepo)
			setOptionalResolverField(previewQ, "ListRepository", listRepo)
		}
	}

	// company/brand usecase
	var companyUC *usecase.CompanyUsecase
	{
		if x, ok := any(cont).(interface {
			CompanyUsecase() *usecase.CompanyUsecase
		}); ok {
			companyUC = x.CompanyUsecase()
		} else if x, ok := any(cont).(interface {
			GetCompanyUsecase() *usecase.CompanyUsecase
		}); ok {
			companyUC = x.GetCompanyUsecase()
		}
	}
	var brandUC *usecase.BrandUsecase
	{
		if x, ok := any(cont).(interface{ BrandUsecase() *usecase.BrandUsecase }); ok {
			brandUC = x.BrandUsecase()
		} else if x, ok := any(cont).(interface{ GetBrandUsecase() *usecase.BrandUsecase }); ok {
			brandUC = x.GetBrandUsecase()
		}
	}

	// ✅ cart usecase（ここが無いと “NewCartHandlerWithQueries” を作れず、未注入になりやすい）
	var cartUC *usecase.CartUsecase
	{
		if x, ok := any(cont).(interface{ CartUsecase() *usecase.CartUsecase }); ok {
			cartUC = x.CartUsecase()
		} else if x, ok := any(cont).(interface{ GetCartUsecase() *usecase.CartUsecase }); ok {
			cartUC = x.GetCartUsecase()
		}
		if cartUC == nil {
			// RouterDeps 側に入っているケースも拾う
			cartUC = getFieldPtr[*usecase.CartUsecase](depsAny, "CartUC", "CartUsecase")
		}
	}

	// core usecases
	listUC := getFieldPtr[*usecase.ListUsecase](depsAny, "ListUC", "ListUsecase")
	invUC := getFieldPtr[*usecase.InventoryUsecase](depsAny, "InventoryUC", "InventoryUsecase")
	pbUC := getFieldPtr[*usecase.ProductBlueprintUsecase](depsAny, "ProductBlueprintUC", "ProductBlueprintUsecase")
	modelUC := getFieldPtr[*usecase.ModelUsecase](depsAny, "ModelUC", "ModelUsecase")
	tokenBlueprintUC := getFieldPtr[*usecase.TokenBlueprintUsecase](depsAny, "TokenBlueprintUC", "TokenBlueprintUsecase")

	snsDeps := NewSNSDepsWithNameResolverAndOrgHandlers(
		listUC, invUC, pbUC, modelUC, tokenBlueprintUC,
		companyUC, brandUC, nameResolver, catalogQ,
	)

	// ✅ sign-in
	snsDeps.SignIn = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSSignInHandler", "GetSNSSignInHandler",
			"SNSSignIn", "GetSNSSignIn",
			"SignInHandler", "GetSignInHandler",
			"SignIn", "GetSignIn",
		},
		[]string{
			"SNSSignInHandler", "SNSSignIn",
			"SignInHandler", "SignIn",
		},
	)

	// ============================================================
	// ✅ auth onboarding (User / Shipping / Billing / Avatar)
	// - 501 の主因は “handler が nil” なので、best-effort + 最後に usecase から組み立てる
	// ============================================================

	// ✅ user
	snsDeps.User = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSUserHandler", "GetSNSUserHandler",
			"SNSUser", "GetSNSUser",
			"SNSUserHandler", "GetSNSUserHandler",
			"SNSUserProfileHandler", "GetSNSUserProfileHandler",
			"UserHandler", "GetUserHandler",
			"User", "GetUser",
		},
		[]string{
			"SNSUserHandler", "SNSUser", "SNSUserHandler",
			"UserHandler", "User",
		},
	)
	if snsDeps.User == nil {
		userUC := getFieldPtr[*usecase.UserUsecase](depsAny, "UserUC", "UserUsecase", "SNSUserUC", "SNSUserUsecase")
		if userUC != nil {
			snsDeps.User = snshandler.NewUserHandler(userUC)
		}
	}

	// ✅ shipping address
	snsDeps.ShippingAddress = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSShippingAddressHandler", "GetSNSShippingAddressHandler",
			"SNSShippingAddress", "GetSNSShippingAddress",
			"ShippingAddressHandler", "GetShippingAddressHandler",
			"ShippingAddress", "GetShippingAddress",
			"SNSShippingHandler", "GetSNSShippingHandler",
			"SNSShipping", "GetSNSShipping",
		},
		[]string{
			"SNSShippingAddressHandler", "SNSShippingAddress",
			"ShippingAddressHandler", "ShippingAddress",
			"SNSShippingHandler", "SNSShipping",
		},
	)
	if snsDeps.ShippingAddress == nil {
		shipUC := getFieldPtr[*usecase.ShippingAddressUsecase](depsAny, "ShippingAddressUC", "ShippingAddressUsecase", "SNSShippingAddressUC", "SNSShippingAddressUsecase")
		if shipUC != nil {
			snsDeps.ShippingAddress = snshandler.NewShippingAddressHandler(shipUC)
		}
	}

	// ✅ billing address（名揺れ吸収）
	snsDeps.BillingAddress = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSBillingAddressHandler", "GetSNSBillingAddressHandler",
			"SNSBillingAddress", "GetSNSBillingAddress",
			"BillingAddressHandler", "GetBillingAddressHandler",
			"BillingAddress", "GetBillingAddress",
			"SNSBillingHandler", "GetSNSBillingHandler",
			"SNSBilling", "GetSNSBilling",
		},
		[]string{
			"SNSBillingAddressHandler", "SNSBillingAddress",
			"BillingAddressHandler", "BillingAddress",
			"SNSBillingHandler", "SNSBilling",
		},
	)
	if snsDeps.BillingAddress == nil {
		billUC := getFieldPtr[*usecase.BillingAddressUsecase](depsAny, "BillingAddressUC", "BillingAddressUsecase", "SNSBillingAddressUC", "SNSBillingAddressUsecase")
		if billUC != nil {
			snsDeps.BillingAddress = snshandler.NewBillingAddressHandler(billUC)
		}
	}

	// ✅ avatar
	snsDeps.Avatar = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSAvatarHandler", "GetSNSAvatarHandler",
			"SNSAvatar", "GetSNSAvatar",
			"AvatarHandler", "GetAvatarHandler",
			"Avatar", "GetAvatar",
		},
		[]string{
			"SNSAvatarHandler", "SNSAvatar",
			"AvatarHandler", "Avatar",
		},
	)
	if snsDeps.Avatar == nil {
		avatarUC := getFieldPtr[*usecase.AvatarUsecase](depsAny, "AvatarUC", "AvatarUsecase", "SNSAvatarUC", "SNSAvatarUsecase")
		if avatarUC != nil {
			snsDeps.Avatar = snshandler.NewAvatarHandler(avatarUC)
		}
	}

	// ✅ avatar state
	snsDeps.AvatarState = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSAvatarStateHandler", "GetSNSAvatarStateHandler",
			"SNSAvatarState", "GetSNSAvatarState",
			"AvatarStateHandler", "GetAvatarStateHandler",
			"AvatarState", "GetAvatarState",
		},
		[]string{
			"SNSAvatarStateHandler", "SNSAvatarState",
			"AvatarStateHandler", "AvatarState",
		},
	)

	// ✅ wallet
	snsDeps.Wallet = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSWalletHandler", "GetSNSWalletHandler",
			"SNSWallet", "GetSNSWallet",
			"WalletHandler", "GetWalletHandler",
			"Wallet", "GetWallet",
		},
		[]string{
			"SNSWalletHandler", "SNSWallet",
			"WalletHandler", "Wallet",
		},
	)

	// ============================================================
	// ✅ cart + preview（重要）
	//
	// 目的:
	// - GET /sns/cart と GET /sns/cart/query は cart_query.go を確実に通す（read-model）
	// - それ以外（POST/PUT/DELETE /sns/cart/*）は cart handler（write）へ委譲
	// ============================================================

	if cartUC != nil {
		core := snshandler.NewCartHandlerWithQueries(cartUC, cartQ, previewQ)

		// ✅ GET /sns/cart, /sns/cart/, /sns/cart/query* だけ cart_query に寄せる
		if cartQ != nil {
			qh := snshandler.NewSNSCartQueryHandler(cartQ)

			wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				p := r.URL.Path

				// GET のみ read-model へ
				if r.Method == http.MethodGet {
					if p == "/sns/cart" || p == "/sns/cart/" || strings.HasPrefix(p, "/sns/cart/query") {
						qh.ServeHTTP(w, r)
						return
					}
				}

				// それ以外は write handler へ
				core.ServeHTTP(w, r)
			})

			snsDeps.Cart = wrapped
		} else {
			snsDeps.Cart = core
		}

		// Preview は既存 handler に任せる（/sns/preview は別登録）
		snsDeps.Preview = core
	} else {
		// fallback: 既存の best-effort を維持（ただし query 未注入になりやすい）
		snsDeps.Cart = getHandlerBestEffort(cont, depsAny,
			[]string{
				"SNSCartHandler", "GetSNSCartHandler",
				"SNSCart", "GetSNSCart",
				"CartHandler", "GetCartHandler",
				"Cart", "GetCart",
			},
			[]string{
				"SNSCartHandler", "SNSCart",
				"CartHandler", "Cart",
			},
		)

		snsDeps.Preview = getHandlerBestEffort(cont, depsAny,
			[]string{
				"SNSPreviewHandler", "GetSNSPreviewHandler",
				"SNSPreview", "GetSNSPreview",
				"PreviewHandler", "GetPreviewHandler",
				"Preview", "GetPreview",
			},
			[]string{
				"SNSPreviewHandler", "SNSPreview",
				"PreviewHandler", "Preview",
			},
		)

		// 片方しか見つからない場合でも、Cart が /sns/preview を捌ける前提なら補完する
		if snsDeps.Preview == nil && snsDeps.Cart != nil {
			snsDeps.Preview = snsDeps.Cart
		}

		// 旧: reflection 注入（exported field の場合のみ効く可能性があるため残す）
		if snsDeps.Cart != nil && cartQ != nil {
			setOptionalResolverField(snsDeps.Cart, "CartQuery", cartQ)
			setOptionalResolverField(snsDeps.Cart, "Query", cartQ)
			setOptionalResolverField(snsDeps.Cart, "CartQ", cartQ)
		}
		if snsDeps.Preview != nil && previewQ != nil {
			setOptionalResolverField(snsDeps.Preview, "PreviewQuery", previewQ)
			setOptionalResolverField(snsDeps.Preview, "Query", previewQ)
			setOptionalResolverField(snsDeps.Preview, "PreviewQ", previewQ)
		}
	}

	// ✅ posts
	snsDeps.Post = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSPostHandler", "GetSNSPostHandler",
			"SNSPost", "GetSNSPost",
			"PostHandler", "GetPostHandler",
			"Post", "GetPost",
		},
		[]string{
			"SNSPostHandler", "SNSPost",
			"PostHandler", "Post",
		},
	)

	// ✅ payment（Container が SNSPaymentHandler() を持っている前提で best-effort で取得）
	snsDeps.Payment = getHandlerBestEffort(cont, depsAny,
		[]string{
			"SNSPaymentHandler", "GetSNSPaymentHandler",
			"SNSPayment", "GetSNSPayment",
			"PaymentHandler", "GetPaymentHandler",
			"Payment", "GetPayment",
		},
		[]string{
			"SNSPaymentHandler", "SNSPayment",
			"PaymentHandler", "Payment",
		},
	)

	// ✅ ここが “確定ログ”
	log.Printf("[sns_container] inject result signIn=%t user=%t ship=%t bill=%t avatar=%t cart=%t preview=%t payment=%t cartUC=%t cartQ=%t previewQ=%t listRepo=%t",
		snsDeps.SignIn != nil,
		snsDeps.User != nil,
		snsDeps.ShippingAddress != nil,
		snsDeps.BillingAddress != nil,
		snsDeps.Avatar != nil,
		snsDeps.Cart != nil,
		snsDeps.Preview != nil,
		snsDeps.Payment != nil,
		cartUC != nil,
		cartQ != nil,
		previewQ != nil,
		listRepo != nil,
	)

	// ✅ 見つからない場合、候補名をログに出す（原因特定用）
	if snsDeps.SignIn == nil {
		log.Printf("[sns_container] SignIn handler not found. candidates=%s", debugHandlerCandidates(cont, depsAny, "signin", "sign", "auth", "login"))
	}
	if snsDeps.User == nil {
		log.Printf("[sns_container] User handler not found. candidates=%s", debugHandlerCandidates(cont, depsAny, "user", "profile"))
	}
	if snsDeps.ShippingAddress == nil {
		log.Printf("[sns_container] ShippingAddress handler not found. candidates=%s", debugHandlerCandidates(cont, depsAny, "shipping", "address"))
	}
	if snsDeps.BillingAddress == nil {
		log.Printf("[sns_container] BillingAddress handler not found. candidates=%s", debugHandlerCandidates(cont, depsAny, "billing", "address"))
	}
	if snsDeps.Avatar == nil {
		log.Printf("[sns_container] Avatar handler not found. candidates=%s", debugHandlerCandidates(cont, depsAny, "avatar"))
	}
	if snsDeps.Cart == nil {
		log.Printf("[sns_container] Cart handler not found. candidates=%s", debugHandlerCandidates(cont, depsAny, "cart"))
	}
	if snsDeps.Preview == nil {
		log.Printf("[sns_container] Preview handler not found. candidates=%s", debugHandlerCandidates(cont, depsAny, "preview"))
	}
	if snsDeps.Payment == nil {
		log.Printf("[sns_container] Payment handler not found. candidates=%s", debugHandlerCandidates(cont, depsAny, "payment", "order", "checkout"))
	}

	// ============================================================
	// ✅ IMPORTANT:
	// net/http.ServeMux は「パスプレフィックスで middleware を束ねる」仕組みがないため、
	// /sns/payment のような buyer-auth 必須ルートは “handler を wrap してから mux に登録” する必要がある。
	// ============================================================
	{
		userAuth := newUserAuthMiddlewareBestEffort(cont.FirebaseAuth)
		if userAuth == nil {
			log.Printf("[sns_container] WARN: user_auth middleware is not available (firebase auth client missing). protected routes may 401.")
		} else {
			wrap := func(h http.Handler) http.Handler {
				if h == nil {
					return nil
				}
				return userAuth.Handler(h)
			}

			// ✅ buyer-auth を要求するものだけ wrap（公開系は wrap しない）
			// NOTE: SignIn は “入口” になり得るので wrap しない（token なしで叩ける余地を残す）
			snsDeps.User = wrap(snsDeps.User)
			snsDeps.ShippingAddress = wrap(snsDeps.ShippingAddress)
			snsDeps.BillingAddress = wrap(snsDeps.BillingAddress)
			snsDeps.Avatar = wrap(snsDeps.Avatar)

			snsDeps.AvatarState = wrap(snsDeps.AvatarState)
			snsDeps.Wallet = wrap(snsDeps.Wallet)
			snsDeps.Cart = wrap(snsDeps.Cart)
			snsDeps.Preview = wrap(snsDeps.Preview)
			snsDeps.Post = wrap(snsDeps.Post)
			snsDeps.Payment = wrap(snsDeps.Payment)

			log.Printf("[sns_container] user_auth applied: user=%t ship=%t bill=%t avatar=%t avatarState=%t wallet=%t cart=%t preview=%t post=%t payment=%t",
				snsDeps.User != nil,
				snsDeps.ShippingAddress != nil,
				snsDeps.BillingAddress != nil,
				snsDeps.Avatar != nil,
				snsDeps.AvatarState != nil,
				snsDeps.Wallet != nil,
				snsDeps.Cart != nil,
				snsDeps.Preview != nil,
				snsDeps.Post != nil,
				snsDeps.Payment != nil,
			)
		}
	}

	RegisterSNSRoutes(mux, snsDeps)
}

// RegisterSNSRoutes registers buyer-facing routes onto mux.
func RegisterSNSRoutes(mux *http.ServeMux, deps SNSDeps) {
	if mux == nil {
		return
	}

	snshttp.Register(mux, snshttp.Deps{
		List:             deps.List,
		Inventory:        deps.Inventory,
		ProductBlueprint: deps.ProductBlueprint,
		Model:            deps.Model,
		Catalog:          deps.Catalog,

		TokenBlueprint: deps.TokenBlueprint,

		Company: deps.Company,
		Brand:   deps.Brand,

		SignIn: deps.SignIn,

		User:            deps.User,
		ShippingAddress: deps.ShippingAddress,
		BillingAddress:  deps.BillingAddress,
		Avatar:          deps.Avatar,

		AvatarState: deps.AvatarState,
		Wallet:      deps.Wallet,
		Cart:        deps.Cart,
		Preview:     deps.Preview,
		Post:        deps.Post,

		Payment: deps.Payment,
	})
}

// ------------------------------------------------------------
// Middleware builder
// ------------------------------------------------------------

// newUserAuthMiddlewareBestEffort builds middleware.UserAuthMiddleware from firebaseauth.Client.
//
// user_auth.go expects *middleware.FirebaseAuthClient.
// In your codebase, middleware.FirebaseAuthClient is a type alias of firebaseauth.Client
// (declared in member_auth.go), so we can pass it directly.
func newUserAuthMiddlewareBestEffort(fb *firebaseauth.Client) *middleware.UserAuthMiddleware {
	if fb == nil {
		return nil
	}
	return &middleware.UserAuthMiddleware{FirebaseAuth: fb}
}

// ------------------------------------------------------------
// Reflection helpers
// ------------------------------------------------------------

func setOptionalResolverField(handler any, fieldName string, value any) {
	if handler == nil || value == nil || strings.TrimSpace(fieldName) == "" {
		return
	}

	rv := reflect.ValueOf(handler)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}

	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() {
		return
	}

	vv := reflect.ValueOf(value)
	if !vv.IsValid() {
		return
	}

	if vv.Type().AssignableTo(f.Type()) {
		f.Set(vv)
		return
	}

	if f.Kind() == reflect.Interface && vv.Type().Implements(f.Type()) {
		f.Set(vv)
		return
	}
}

func getSNSNameResolverFieldBestEffort(cont *Container) *appresolver.NameResolver {
	if cont == nil {
		return nil
	}

	rv := reflect.ValueOf(cont)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	tryField := func(name string) *appresolver.NameResolver {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			return nil
		}
		if f.Kind() == reflect.Interface {
			if f.IsNil() {
				return nil
			}
			v := f.Interface()
			if nr, ok := v.(*appresolver.NameResolver); ok {
				return nr
			}
			return nil
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				return nil
			}
			if nr, ok := f.Interface().(*appresolver.NameResolver); ok {
				return nr
			}
			return nil
		}
		return nil
	}

	for _, n := range []string{
		"SNSNameResolver",
		"SnsNameResolver",
		"snsNameResolver",
		"NameResolver",
		"nameResolver",
	} {
		if nr := tryField(n); nr != nil {
			return nr
		}
	}

	return nil
}

func getSNSCartQueryFieldBestEffort(cont *Container) *snsquery.SNSCartQuery {
	if cont == nil {
		return nil
	}

	rv := reflect.ValueOf(cont)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	tryField := func(name string) *snsquery.SNSCartQuery {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			return nil
		}
		if f.Kind() == reflect.Interface {
			if f.IsNil() {
				return nil
			}
			if q, ok := f.Interface().(*snsquery.SNSCartQuery); ok {
				return q
			}
			return nil
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				return nil
			}
			if q, ok := f.Interface().(*snsquery.SNSCartQuery); ok {
				return q
			}
			return nil
		}
		return nil
	}

	for _, n := range []string{
		"SNSCartQuery",
		"SNSCartQ",
		"SnsCartQuery",
		"snsCartQuery",
		"snsCartQ",
		"CartQuery",
		"CartQ",
		"cartQuery",
		"cartQ",
	} {
		if q := tryField(n); q != nil {
			return q
		}
	}

	return nil
}

func getSNSPreviewQueryFieldBestEffort(cont *Container) *snsquery.SNSPreviewQuery {
	if cont == nil {
		return nil
	}

	rv := reflect.ValueOf(cont)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	tryField := func(name string) *snsquery.SNSPreviewQuery {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			return nil
		}
		if f.Kind() == reflect.Interface {
			if f.IsNil() {
				return nil
			}
			if q, ok := f.Interface().(*snsquery.SNSPreviewQuery); ok {
				return q
			}
			return nil
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				return nil
			}
			if q, ok := f.Interface().(*snsquery.SNSPreviewQuery); ok {
				return q
			}
			return nil
		}
		return nil
	}

	for _, n := range []string{
		"SNSPreviewQuery",
		"SNSPreviewQ",
		"SnsPreviewQuery",
		"snsPreviewQuery",
		"snsPreviewQ",
		"PreviewQuery",
		"PreviewQ",
		"previewQuery",
		"previewQ",
	} {
		if q := tryField(n); q != nil {
			return q
		}
	}

	return nil
}

func getFieldPtr[T any](src any, names ...string) T {
	var zero T
	if src == nil {
		return zero
	}

	rv := reflect.ValueOf(src)
	if !rv.IsValid() {
		return zero
	}
	if rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return zero
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return zero
	}

	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		f := rv.FieldByName(n)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}
		if v, ok := f.Interface().(T); ok {
			return v
		}
	}

	return zero
}

func normName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	// getter prefix
	s = strings.TrimPrefix(s, "get")
	// common suffix
	s = strings.TrimSuffix(s, "handler")
	return s
}

func nameMatches(candidate string, targets []string) bool {
	cn := normName(candidate)
	for _, t := range targets {
		if cn == normName(t) {
			return true
		}
	}
	return false
}

// getHandlerBestEffort finds a handler from Container (methods/fields) or RouterDeps fields, best-effort.
// - exact + "Get*" variants
// - case-insensitive-ish (normalize: get/handler/_/- removed)
func getHandlerBestEffort(cont *Container, depsAny any, containerNames []string, depsFieldNames []string) http.Handler {
	// 1) container methods (exact + normalized)
	if cont != nil {
		rv := reflect.ValueOf(cont)
		rt := rv.Type()

		for i := 0; i < rt.NumMethod(); i++ {
			m := rt.Method(i)
			if !nameMatches(m.Name, containerNames) {
				continue
			}
			mv := rv.MethodByName(m.Name)
			if !mv.IsValid() {
				continue
			}
			if mv.Type().NumIn() != 0 || mv.Type().NumOut() != 1 {
				continue
			}
			out := mv.Call(nil)
			if len(out) != 1 {
				continue
			}
			if h, ok := out[0].Interface().(http.Handler); ok {
				return h
			}
		}

		// 2) container fields (normalized match)
		rve := reflect.ValueOf(cont)
		if rve.IsValid() && rve.Kind() == reflect.Pointer && !rve.IsNil() {
			rve = rve.Elem()
		}
		if rve.IsValid() && rve.Kind() == reflect.Struct {
			rtf := rve.Type()
			for i := 0; i < rtf.NumField(); i++ {
				f := rtf.Field(i)
				if !nameMatches(f.Name, containerNames) {
					continue
				}
				fv := rve.Field(i)
				if !fv.IsValid() || !fv.CanInterface() {
					continue
				}
				if h, ok := fv.Interface().(http.Handler); ok {
					return h
				}
			}
		}
	}

	// 3) deps fields
	if depsAny != nil {
		rv := reflect.ValueOf(depsAny)
		if rv.Kind() == reflect.Interface && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				return nil
			}
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Struct {
			rt := rv.Type()
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				if !nameMatches(f.Name, depsFieldNames) {
					continue
				}
				fv := rv.Field(i)
				if !fv.IsValid() || !fv.CanInterface() {
					continue
				}
				if h, ok := fv.Interface().(http.Handler); ok {
					return h
				}
			}
		}
	}

	return nil
}

// debugHandlerCandidates dumps "it exists but name mismatch" vs "doesn't exist at all".
func debugHandlerCandidates(cont *Container, depsAny any, keywords ...string) string {
	kw := make([]string, 0, len(keywords))
	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" {
			kw = append(kw, k)
		}
	}
	contains := func(name string) bool {
		n := strings.ToLower(name)
		for _, k := range kw {
			if strings.Contains(n, k) {
				return true
			}
		}
		return false
	}

	out := make([]string, 0, 32)

	// container methods
	if cont != nil {
		rv := reflect.ValueOf(cont)
		rt := rv.Type()
		for i := 0; i < rt.NumMethod(); i++ {
			m := rt.Method(i)
			if !contains(m.Name) {
				continue
			}
			mv := rv.MethodByName(m.Name)
			if !mv.IsValid() {
				continue
			}
			// zero-arg, one-out, and out implements http.Handler
			if mv.Type().NumIn() == 0 && mv.Type().NumOut() == 1 {
				ot := mv.Type().Out(0)
				if ot.Implements(reflect.TypeOf((*http.Handler)(nil)).Elem()) {
					out = append(out, "Container.method:"+m.Name)
				}
			}
		}

		// container fields
		rve := reflect.ValueOf(cont)
		if rve.IsValid() && rve.Kind() == reflect.Pointer && !rve.IsNil() {
			rve = rve.Elem()
		}
		if rve.IsValid() && rve.Kind() == reflect.Struct {
			rtf := rve.Type()
			for i := 0; i < rtf.NumField(); i++ {
				f := rtf.Field(i)
				if !contains(f.Name) {
					continue
				}
				fv := rve.Field(i)
				if fv.IsValid() && fv.CanInterface() {
					if _, ok := fv.Interface().(http.Handler); ok {
						out = append(out, "Container.field:"+f.Name)
					}
				}
			}
		}
	}

	// deps fields
	if depsAny != nil {
		rv := reflect.ValueOf(depsAny)
		if rv.Kind() == reflect.Interface && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			rt := rv.Type()
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				if !contains(f.Name) {
					continue
				}
				fv := rv.Field(i)
				if fv.IsValid() && fv.CanInterface() {
					if _, ok := fv.Interface().(http.Handler); ok {
						out = append(out, "RouterDeps.field:"+f.Name)
					}
				}
			}
		}
	}

	sort.Strings(out)
	if len(out) == 0 {
		return "(none)"
	}
	// 長すぎ防止
	if len(out) > 24 {
		out = out[:24]
		out = append(out, "...(truncated)")
	}
	return strings.Join(out, ", ")
}

// ------------------------------------------------------------
// Firestore resolver (best-effort)
// ------------------------------------------------------------

// getFirestoreClientBestEffort tries to obtain *firestore.Client from Container or RouterDeps.
// This is used to reliably construct ListRepo here (so listRepo=false does not happen).
func getFirestoreClientBestEffort(cont *Container, depsAny any) *firestore.Client {
	// 1) Container methods
	if cont != nil {
		if x, ok := any(cont).(interface{ Firestore() *firestore.Client }); ok {
			if fs := x.Firestore(); fs != nil {
				return fs
			}
		}
		if x, ok := any(cont).(interface{ FirestoreClient() *firestore.Client }); ok {
			if fs := x.FirestoreClient(); fs != nil {
				return fs
			}
		}
		if x, ok := any(cont).(interface{ FS() *firestore.Client }); ok {
			if fs := x.FS(); fs != nil {
				return fs
			}
		}
		if x, ok := any(cont).(interface{ GetFirestore() *firestore.Client }); ok {
			if fs := x.GetFirestore(); fs != nil {
				return fs
			}
		}
		if x, ok := any(cont).(interface{ GetFirestoreClient() *firestore.Client }); ok {
			if fs := x.GetFirestoreClient(); fs != nil {
				return fs
			}
		}

		// 2) Container fields (reflection)
		rv := reflect.ValueOf(cont)
		if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			for _, n := range []string{"Firestore", "FirestoreClient", "FS", "Client"} {
				f := rv.FieldByName(n)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				if fs, ok := f.Interface().(*firestore.Client); ok && fs != nil {
					return fs
				}
			}
		}
	}

	// 3) RouterDeps fields
	if depsAny != nil {
		rv := reflect.ValueOf(depsAny)
		if rv.Kind() == reflect.Interface && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			for _, n := range []string{"Firestore", "FirestoreClient", "FS", "Client"} {
				f := rv.FieldByName(n)
				if !f.IsValid() || !f.CanInterface() {
					continue
				}
				if fs, ok := f.Interface().(*firestore.Client); ok && fs != nil {
					return fs
				}
			}
		}
	}

	return nil
}
