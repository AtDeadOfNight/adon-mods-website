export function Mod({ name, description, link, image, requiresSignIn = false }: { name: string, description: string, image: string, link: string, requiresSignIn?: boolean }) {
  return (
    <div className='w-full p-6 rounded-xl bg-[#2c2c2c] border-[0.5px] border-[rgba(255,255,255,0.25)] shadow-md flex flex-col gap-3 items-start'>
      <h3 className='my-0 font-semibold text-2xl leading-tight 1124px:text-3xl 1124px:leading-normal'>{name}</h3>
      <p className='mb-2 leading-snug text-sm 1124px:text-base'>{description}</p>
      <img src={"/assets/" + image} className='w-full rounded-md mb-4 border border-neutral-800 shadow-md bg-neutral-300 mt-auto'></img>
      <div className='flex gap-4 items-start'>
        <a href={link} className='bg-indigo-600 text-white py-1.5 px-3 rounded-md hover:text-neutral-200 transition-colors duration-200 shadow-lg' target='_blank' rel='noreferrer noopener'>Download</a>
        {requiresSignIn && <span className='text-xs leading-tight text-neutral-500'>This download requires signing in with Steam to confirm you're owning At Dead Of Night copy</span>}
      </div>
    </div>
  )
}