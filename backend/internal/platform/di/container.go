// internal/platform/di/container.go
package di

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	// プラットフォーム系
	"narratives/internal/platform/config"

	// アウトバウンドポート（インターフェース）
	"narratives/internal/ports/outbound"

	// アウトバウンドアダプタ実装（例: PG, Firestore, FirebaseAdmin 等）
	// ここはあなたの実装パッケージ名に合わせて import してください。
	authadapter "narratives/internal/adapters/outbound/auth" // 仮: Firebase Admin 等
	pgrepo "narratives/internal/adapters/outbound/db/repository"

	// maileradapter "narratives/internal/adapters/outbound/notify/mailer"
	// etc...

	// アプリケーション層ユースケース
	memberuc "narratives/internal/application/usecase"

	// インバウンドアダプタ (HTTPハンドラ)
	adminhttp "narratives/internal/adapters/inbound/http/admin"
	// authhttp は /admin/users/bootstrap のような初期化APIハンドラをまとめる想定
	authhttp "narratives/internal/adapters/inbound/http/auth" // ←あなた側で用意する前提

	// GraphQL ハンドラ（gqlgen等で生成された http.Handler をラップしたもの）
	graphhttp "narratives/internal/adapters/inbound/http/graphql"

	// 必要ならDBドライバ
	_ "github.com/lib/pq"
)

// Container は main.go から使う依存オブジェクトの束。
// これを返したい目的は：main.go を極限まで薄くすること。
type Container struct {
	// GraphQLエンドポイント用 http.Handler
	GraphQLHandler graphhttp.Handler // あなたの実装に合わせて http.Handler でもOK

	// RESTエンドポイント群
	REST struct {
		Admin *adminhttp.MembersHandler // /admin/members/... /admin/iam/...
		Auth  *authhttp.AuthHandler     // /admin/users/bootstrap 等、管理者用ブートストラップAPI
	}

	// 下層リソースを閉じるためのコンテキスト
	db        *sql.DB
	auth      outbound.AuthProvider
	cleanupFn []func()
}

// Close は Cloud Run 終了時などに呼んで安全にリソースを閉じる。
func (c *Container) Close() {
	if c.db != nil {
		_ = c.db.Close()
	}
	for _, fn := range c.cleanupFn {
		fn()
	}
}

// Build は DIコンテナを初期化して返す。
// - 環境変数/設定の読み込み済み cfg をもらう
// - DB接続や外部クライアントを組み立てる
// - Repository実装とUsecaseとHandlerを全部つなぐ
func Build(cfg *config.Config) (*Container, func(), error) {
	ctx := context.Background()

	// ------------------------------------------------------------
	// 1. 外部リソース初期化 (DB / Auth / Firestore / Redis / etc.)
	// ------------------------------------------------------------

	// ※ Postgres 等のRDB
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("open db: %w", err)
	}
	// DBプール設定など（必要に応じて）
	// db.SetMaxOpenConns(...)
	// db.SetConnMaxLifetime(...)

	// Pingで疎通チェック
	if pingErr := db.PingContext(ctx); pingErr != nil {
		log.Printf("[di] WARN: db ping failed: %v", pingErr)
		// ここで return してもいいし、"db無しモード"許容なら続行する。
	}

	// ※ Firebase Admin / AuthProvider
	//   - ここでは "authadapter.NewFirebaseAuthProvider(cfg)" のような
	//     あなた側で用意予定のコンストラクタを想定。
	authProv, err := authadapter.NewFirebaseAuthProvider(ctx, cfg.FirebaseCredJSONPath)
	if err != nil {
		log.Printf("[di] WARN: failed to init AuthProvider, fallback to no-auth: %v", err)
		// fallback用のダミー実装でも可
		// authProv = authadapter.NewNoopAuthProvider()
		return nil, nil, fmt.Errorf("init auth provider: %w", err)
	}

	// （必要なら）Firestore, Redis, GCS, SendGrid などもここで初期化して
	// cleanupFn に Close() を積んでいく

	// cleanup 関数たちをまとめる
	cleanupList := []func(){}
	cleanupList = append(cleanupList, func() {
		// 例: FirestoreClient.Close() とか、外部コネクションの解放
	})

	// ------------------------------------------------------------
	// 2. Repository (outbound adapter) を初期化
	// ------------------------------------------------------------
	//
	// MemberRepository は Firestore or Postgres 側の実装など、
	// あなたの実態に合わせて選ぶ。
	//
	// ここでは例として Postgres の実装を使うイメージ:
	memberRepo := pgrepo.NewMemberRepositoryPG(db)
	// ↑ 実装名はあなたのリポジトリ名に合わせて修正してください。
	// Firestoreベースなら Firestore用の NewMemberRepositoryFS(...) を呼ぶ。

	// ------------------------------------------------------------
	// 3. Usecase を初期化
	// ------------------------------------------------------------

	inviteUC := &memberuc.InviteMember{
		Auth: authProv,
		Repo: memberRepo,
	}
	updateRolesUC := &memberuc.UpdateMemberRoles{
		Auth: authProv,
		Repo: memberRepo,
	}
	deleteMemberUC := &memberuc.DeleteMember{
		Auth: authProv,
		Repo: memberRepo,
	}
	elevateRootUC := &memberuc.ElevateRootOnSignIn{
		Auth: authProv,
		Repo: memberRepo,
	}
	deleteByEmailUC := &memberuc.AdminDeleteUserByEmail{
		Auth: authProv,
		Repo: memberRepo,
	}

	// ------------------------------------------------------------
	// 4. Inbound HTTP Handler を初期化
	// ------------------------------------------------------------

	// /admin/members/... を扱うハンドラ
	adminMembersHandler := &adminhttp.MembersHandler{
		InviteUC:        inviteUC,
		UpdateUC:        updateRolesUC,
		DeleteUC:        deleteMemberUC,
		ElevateUC:       elevateRootUC,
		DeleteByEmailUC: deleteByEmailUC,
	}

	// /admin/users/bootstrap 等を扱う別の管理系ハンドラ
	// これはあなたの側で AuthHandler (or BootstrapHandler) 的なものを
	// internal/adapters/inbound/http/auth/ に用意しておく前提。
	adminAuthHandler := authhttp.NewAuthHandler(authhttp.AuthHandlerDeps{
		AuthProvider: authProv,
		MemberRepo:   memberRepo,
		// ほか必要ならここで注入
	})

	// GraphQL ハンドラ
	// gqlgen等で生成された http.Handler をラップした c.GraphQLHandler を想定。
	// graphhttp.NewGraphQLHandler(...) はあなた側で用意している前提。
	gqlHandler := graphhttp.NewGraphQLHandler(graphhttp.Deps{
		// ここに Usecase や Repo を注入して resolver が使えるようにする
		MemberRepo: memberRepo,
		Auth:       authProv,
	})

	// ------------------------------------------------------------
	// 5. Container を組み立てて返す
	// ------------------------------------------------------------

	container := &Container{
		GraphQLHandler: gqlHandler,
		db:             db,
		auth:           authProv,
		cleanupFn:      cleanupList,
	}
	container.REST.Admin = adminMembersHandler
	container.REST.Auth = adminAuthHandler

	cleanup := func() {
		container.Close()
	}

	return container, cleanup, nil
}
