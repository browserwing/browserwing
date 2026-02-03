import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { ThemeProvider } from './contexts/ThemeContext'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import BrowserManager from './pages/BrowserManager'
import BrowserInstanceManager from './pages/BrowserInstanceManager'
import CookieManager from './pages/CookieManager'
import ScriptManager from './pages/ScriptManager'
import LLMManager from './pages/LLMManager'
import PromptManage from './pages/PromptManage'
import AgentChat from './pages/AgentChat'
import ToolManager from './pages/ToolManager'
import Login from './pages/Login'
import Settings from './pages/Settings'
import ScheduledTaskManager from './pages/ScheduledTaskManager'
import { checkAuth } from './api/client'

// 受保护的路由组件
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const [authEnabled, setAuthEnabled] = useState<boolean | null>(null)
  const [loading, setLoading] = useState(true)
  const location = useLocation()

  useEffect(() => {
    const checkAuthentication = async () => {
      try {
        const enabled = await checkAuth()
        setAuthEnabled(enabled)
      } catch (error) {
        setAuthEnabled(false)
      } finally {
        setLoading(false)
      }
    }

    checkAuthentication()
  }, [])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-600 dark:text-gray-400">Loading...</div>
      </div>
    )
  }

  // 如果未启用认证，直接显示内容
  if (!authEnabled) {
    return <>{children}</>
  }

  // 如果启用了认证，检查是否有token
  const token = localStorage.getItem('token')
  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <>{children}</>
}

function App() {
  return (
    <ThemeProvider>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Dashboard />} />
          <Route path="browser" element={<BrowserManager />} />
          <Route path="browser/instances" element={<BrowserInstanceManager />} />
          <Route path="cookies" element={<CookieManager />} />
          <Route path="scripts" element={<ScriptManager />} />
          <Route path="scheduled-tasks" element={<ScheduledTaskManager />} />
          <Route path="llm" element={<LLMManager />} />
          <Route path="prompts" element={<PromptManage />} />
          <Route path="agent" element={<AgentChat />} />
          <Route path="tools" element={<ToolManager />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
    </ThemeProvider>
  )
}

export default App

