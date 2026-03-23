import { Card, Typography, Space } from 'antd'

const { Title, Paragraph } = Typography

function Home() {
  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card>
        <Title level={2}>Welcome to Agentic Coding Platform</Title>
        <Paragraph>
          A universal agent task execution platform that wraps Claude Code CLI
          to provide a managed execution environment for coding tasks.
        </Paragraph>
      </Card>

      <Card title="Quick Start">
        <Paragraph>
          This is a hello-world page. The platform will support:
        </Paragraph>
        <ul>
          <li>Task template management</li>
          <li>Task execution and monitoring</li>
          <li>Sandbox environment management</li>
          <li>Human intervention mechanisms</li>
        </ul>
      </Card>
    </Space>
  )
}

export default Home
