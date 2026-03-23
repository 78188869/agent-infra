import { Routes, Route } from 'react-router-dom'
import { Layout } from 'antd'
import Home from './pages/Home'

const { Header, Content } = Layout

function App() {
  return (
    <Layout className="app-layout">
      <Header style={{ display: 'flex', alignItems: 'center', background: '#001529' }}>
        <div style={{ color: '#fff', fontSize: '18px', fontWeight: 'bold' }}>
          Agentic Coding Platform
        </div>
      </Header>
      <Content className="app-content">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/user/*" element={<Home />} />
          <Route path="/admin/*" element={<Home />} />
        </Routes>
      </Content>
    </Layout>
  )
}

export default App
