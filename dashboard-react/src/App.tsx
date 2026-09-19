import { useEffect } from 'react'
export default function App() {
  useEffect(() => { document.title = 'MyApi · 网关控制台' }, [])
  return <iframe className="legacy-dashboard" title="MyApi 网关控制台" src="/dashboard/legacy/index.html" />
}
