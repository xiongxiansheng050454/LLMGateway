import { useState } from 'react'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import backendStructure from '../../../docs/backend-structure.md?raw'
import apiRequirements from '../../../docs/api-requirements.md?raw'

const docs = [{ title: '后端结构', path: 'docs/backend-structure.md', body: backendStructure }, { title: 'API 需求', path: 'docs/api-requirements.md', body: apiRequirements }]
export function DocsPage() { const [active, setActive] = useState(0); const doc = docs[active]; const html = DOMPurify.sanitize(marked.parse(doc.body) as string); return <div className="bento-grid"><section className="panel span-4"><div className="panel-head"><div><h3>文档目录</h3><p className="muted">点击切换</p></div></div><div className="doc-list">{docs.map((item, index) => <button className={index === active ? 'active' : ''} onClick={() => setActive(index)} key={item.path}>{item.title}</button>)}</div></section><section className="panel span-8"><div className="panel-head"><div><h3>{doc.title}</h3><p className="mono">{doc.path}</p></div></div><article className="markdown-body" dangerouslySetInnerHTML={{ __html: html }} /></section></div> }
