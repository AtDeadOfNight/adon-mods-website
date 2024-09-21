import '@/shared/styles/index.css'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { HomePage } from '../pages'
import { NotFoundPage } from '@/pages/not-found'
import { PurchasePage } from '@/pages/purchase'
import { Toaster } from 'sonner'
import { DownloadPage } from '@/pages/download-page'

const router = createBrowserRouter([
  {
    path: "/",
    element: <HomePage />,
  },
  {
    path: '/purchase',
    element: <PurchasePage />,
  },
  {
    path: '/download-page',
    element: <DownloadPage />,
  },
  {
    path: "*",
    element: <NotFoundPage />,
  },
]);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
    <Toaster richColors />
  </StrictMode>,
)
