
import './App.css'

function App() {
  return (
    <main className="console-layout">
      <header className="console-header">
        <h1 className="console-title">Narratives Console</h1>
        <span className="console-subtitle">お問い合わせ管理</span>
      </header>

      <section className="console-section">
        <div className="console-card">
          <h2>ステータス</h2>
          <ul>
            <li>未対応: 5件</li>
            <li>対応中: 2件</li>
            <li>完了: 18件</li>
          </ul>
        </div>

        <div className="console-card">
          <h2>アクション</h2>
          <button className="console-button">新しい問い合わせを作成</button>
          <button className="console-button secondary">エクスポート</button>
        </div>
      </section>

      <section className="console-table">
        <h2>最近の問い合わせ</h2>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>顧客名</th>
              <th>件名</th>
              <th>状態</th>
              <th>更新日時</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>#1024</td>
              <td>山田 太郎</td>
              <td>請求内容の確認</td>
              <td className="badge badge-open">未対応</td>
              <td>2025/10/28 09:12</td>
            </tr>
            <tr>
              <td>#1023</td>
              <td>佐藤 花子</td>
              <td>アカウント凍結解除</td>
              <td className="badge badge-progress">対応中</td>
              <td>2025/10/27 18:45</td>
            </tr>
            <tr>
              <td>#1022</td>
              <td>鈴木 次郎</td>
              <td>契約更新について</td>
              <td className="badge badge-done">完了</td>
              <td>2025/10/27 14:03</td>
            </tr>
          </tbody>
        </table>
      </section>
    </main>
  );
}

export default App
