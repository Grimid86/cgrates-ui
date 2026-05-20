import { Outlet } from 'react-router-dom'
export default function Layout() {
  return <div className="min-h-screen bg-gray-50"><header className="bg-gray-900 text-white p-4"><h1>Admin OSS</h1></header><main className="p-4"><Outlet /></main></div>
}
