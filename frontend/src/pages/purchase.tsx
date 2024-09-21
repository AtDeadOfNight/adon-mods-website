import { toast } from "sonner"

export function PurchasePage() {
  let signIn = '/auth/steam'

  const file = new URLSearchParams(window.location.search).get('file')
  if (file) {
    signIn += '?file=' + encodeURIComponent(file)
  }

  if (new URLSearchParams(window.location.search).get('success') === 'false') {
    toast.error('You must own At Dead Of Night in Steam to download this mod.')
  }
  
  return (
    <main className='flex flex-col items-center justify-center px-16 text-center'>
      <section className='w-full max-w-[600px] flex flex-col gap-6 items-center'>
        <p className='text-xl font-bold'>To download this mod, you must own <a href="https://store.steampowered.com/app/1450830">At Dead Of Night in Steam</a>.</p>
        <p className='font-semibold'>Please use button below to authorize with your steam account to prove you have a purchased copy of the game.</p>
        <p className='text-sm text-neutral-600'>Don't worry, you won't have to enter your Steam credentials. We use OpenID authentification. We never receive any private info from your account, as stated on Steam authorization page. We also do not store any information about you. We only perform one-time check if you have At Dead Of Night in your library and you won't have to authorize again in order to download other mods that require purchase in Steam.</p>
        <a href={signIn} className='mt-5'><img src="https://community.akamai.steamstatic.com/public/images/signinthroughsteam/sits_01.png" alt="Sign In Button" /></a>
      </section>
    </main>
  )
}