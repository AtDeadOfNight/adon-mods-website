import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <main className='flex flex-col gap-6 items-center justify-center'>
      <h1 className='font-bold'>Page Not Found</h1>
      <Link to='/'>Go back to home</Link>
    </main>
  )
}