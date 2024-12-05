import { Mod } from '@/entity/mod'

export function Mods() {
  return (
    <section className='bg-neutral-800 flex justify-center p-6 md:p-10'>
      <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 w-full max-w-[1200px] gap-6 1280px:gap-8'>
        <Mod
          name='MacOS port'
          image='adon-macos.gif'
          description='At Dead Of Night.app running natively on mac, no Wine or emulators'
          link={'/download?file=' + encodeURIComponent('At Dead Of Night.app')} 
          requiresSignIn
        />
        <Mod
          name='Cheats: radar, items list, wallhack'
          image='cheats.jpeg'
          description={'At Dead Of Night cheats. Radar (map with Jimmy\'s location). Items list. Wallhack (WIP)'}
          link='https://github.com/AtDeadOfNight/cheats'
        />
        <Mod
          name='Russian dubbing'
          image='ru-dub.jpg'
          description='At Dead Of Night with Russian voiceover. Мод для At Dead Of Night с русской озвучкой.'
          link='https://adon.hloth.dev'
        />
        <Mod
          name='At Dead Of Jerma'
          image='adon-jerma.png'
          description='Mod that replaces Jimmy with Jerma985 streamer'
          link='https://github.com/AtDeadOfNight/at-dead-of-jerma-mod'
        />
        <Mod
          name='At Dead Of deez nuts'
          image='atdeadofdeeznuts.png'
          description='Mod by Maxwell Heller (maxwellheller98@gmail.com)'
          link={'/download?file=' + encodeURIComponent('atdeadofdeeznuts.zip')}
          requiresSignIn
        />
      </div>
    </section>
  )
}