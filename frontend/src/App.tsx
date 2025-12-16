import { Routes, Route } from 'react-router-dom'
import { ThemeProvider } from './contexts/ThemeContext'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import BrowserManager from './pages/BrowserManager'
import CookieManager from './pages/CookieManager'
import ScriptManager from './pages/ScriptManager'
import LLMManager from './pages/LLMManager'
import PromptManage from './pages/PromptManage'
import AgentChat from './pages/AgentChat'
import ToolManager from './pages/ToolManager'

function App() {
  return (
    <ThemeProvider>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="browser" element={<BrowserManager />} />
          <Route path="cookies" element={<CookieManager />} />
          <Route path="scripts" element={<ScriptManager />} />
          <Route path="llm" element={<LLMManager />} />
          <Route path="prompts" element={<PromptManage />} />
          <Route path="agent" element={<AgentChat />} />
          <Route path="tools" element={<ToolManager />} />
        </Route>
      </Routes>
    </ThemeProvider>
  )
}

export default App

