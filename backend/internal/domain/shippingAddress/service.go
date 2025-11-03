package shippingAddress

// Shipping Address Service - 配送先住所ビジネスロジック層
// 表示フォーマットや簡易ロジックを定義

// Note: ShippingAddress は同パッケージのエンティティ定義を参照します。

// FormatZipCode は郵便番号を 〒XXXX の形式で返します（未設定時は空文字）。
func FormatZipCode(addr *ShippingAddress) string {
    if addr == nil || addr.ZipCode == "" {
        return ""
    }
    return "〒" + addr.ZipCode
}

// FormatStateCity は「都道府県 市区町村」の形式で返します（どちらか欠ける場合は存在する方のみ）。
func FormatStateCity(addr *ShippingAddress) string {
    if addr == nil {
        return ""
    }
    if addr.State != "" && addr.City != "" {
        return addr.State + " " + addr.City
    }
    if addr.State != "" {
        return addr.State
    }
    if addr.City != "" {
        return addr.City
    }
    return ""
}

// FormatStreet は番地（Street）を返します（未設定時は空文字）。
func FormatStreet(addr *ShippingAddress) string {
    if addr == nil {
        return ""
    }
    return addr.Street
}

// FormatCountry は国名を返します（未設定時は空文字）。
func FormatCountry(addr *ShippingAddress) string {
    if addr == nil {
        return ""
    }
    return addr.Country
}

// IsShippingAddressAvailable は表示可能かを返します。
func IsShippingAddressAvailable(addr *ShippingAddress, loading bool) bool {
    return !loading && addr != nil
}

// IsShippingAddressNotFound は取得できなかったかを返します。
func IsShippingAddressNotFound(addr *ShippingAddress, loading bool) bool {
    return !loading && addr == nil
}

// IsLoading はローディング中かを返します。
func IsLoading(loading bool) bool {
    return loading
}

// GetErrorMessage はエラーメッセージを返します。
func GetErrorMessage() string {
    return "配送先住所の詳細情報を取得できませんでした"
}

// FormatShippingAddressID は表示用の配送先住所ID文字列を返します。
func FormatShippingAddressID(id string) string {
    return "配送先住所ID: " + id
}

// GetLoadingMessage はローディングメッセージを返します。
func GetLoadingMessage() string {
    return "読み込み中..."
}
