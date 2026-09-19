import type { FormEvent, ReactNode } from 'react'

export function Modal({ title, children, submitText, onClose, onSubmit, wide = false }: { title: string; children: ReactNode; submitText?: string; onClose: () => void; onSubmit?: (event: FormEvent<HTMLFormElement>) => void | Promise<void>; wide?: boolean }) {
  const content = <><div className="modal-title"><h3>{title}</h3><button type="button" className="close" onClick={onClose} aria-label="关闭">×</button></div><div className="modal-body">{children}</div><div className="modal-actions"><button type="button" className="button ghost" onClick={onClose}>取消</button>{onSubmit && <button className="button primary" type="submit">{submitText || '保存'}</button>}</div></>
  const className = `modal${wide ? ' modal-wide' : ''}`
  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    return onSubmit?.(event)
  }
  return <div className="modal-backdrop" onMouseDown={event => { if (event.target === event.currentTarget) onClose() }}>{onSubmit ? <form className={className} onSubmit={handleSubmit}>{content}</form> : <div className={className}>{content}</div>}</div>
}
