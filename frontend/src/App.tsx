import { CatalogPage } from './pages/CatalogPage'

function App() {
  return (
    <div className="min-h-screen bg-stone-100">
      <header className="border-b border-stone-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center gap-3 px-4 py-4">
          <span className="text-2xl">🏔️</span>
          <div>
            <h1 className="text-xl font-bold text-stone-800">Mountain Breath</h1>
            <p className="text-xs text-stone-400">
              tea · coffee · honey from the mountains
            </p>
          </div>
        </div>
      </header>

      <CatalogPage />
    </div>
  )
}

export default App
