import React from 'react'

export function DownloadPage() {
  const file = new URLSearchParams(window.location.search).get('file')
  let redirect = '/'
  if (file) {
    redirect = '/download?file=' + encodeURIComponent(file)
  }

  React.useEffect(() => {
    window.location.href = redirect
  }, [])

  return (
    <div className='flex items-center justify-center text-white'>
      <a href={redirect} className='text-white underline underline-offset-2'>
        Click here if your download hasn't started
      </a>
    </div>
  )
}