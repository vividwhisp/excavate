import { useState } from 'react'
import type { FormEvent } from 'react'

export default function SearchInput({
  onSubmit,
  disabled,
}: {
  onSubmit: (query: string) => void
  disabled: boolean
}) {
  const [value, setValue] = useState('')

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const q = value.trim()
    if (!q || disabled) return
    setValue('')
    onSubmit(q)
  }

  return (
    <form className="search-input" onSubmit={submit}>
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="Ask anything…"
        disabled={disabled}
        autoFocus
      />
      <button className="btn-ask" type="submit" disabled={disabled || !value.trim()}>
        Ask
      </button>
    </form>
  )
}