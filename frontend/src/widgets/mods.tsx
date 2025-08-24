import { Mod } from '@/entity/mod';

export function Mods() {
	return (
		<section className="bg-neutral-800 flex justify-center p-6 md:p-10">
			<div
				className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 w-full
					max-w-[1200px] gap-6 1280px:gap-8"
			>
				<Mod
					name="Android port (APK)"
					nameRu="Порт на Android (APK)"
					image="adon-android.png"
					description='Unofficial At Dead Of Night port for Android. ONLY FOR PHONES WITH AT LEAST 8 GB OF RAM. <a href="https://docs.google.com/spreadsheets/d/1oyyau5VPeA4obLWvlPKQQSB8kgIYfQoX7xH1DXmIr6A/edit?usp=sharing" target="_blank" rel="noreferrer noopener">Check if your device is compatible</a>'
					descriptionRu='Неофициальный порт At Dead Of Night на Android. ТОЛЬКО ДЛЯ УСТРОЙСТВ С 8 ГБ ОПЕРАТИВНОЙ ПАМЯТИ И БОЛЕЕ. <a href="https://docs.google.com/spreadsheets/d/1oyyau5VPeA4obLWvlPKQQSB8kgIYfQoX7xH1DXmIr6A/edit?usp=sharing" target="_blank" rel="noreferrer noopener">Проверьте совместимость вашего устройства</a>'
					link={
						'/download?file=' +
						encodeURIComponent(
							'at-dead-of-night-android-v1.0.0-t.me_adonru.apk'
						)
					}
					requiresSignIn
				/>
				<Mod
					name="iPhone port (IPA)"
					nameRu="Порт на iPhone (IPA)"
					image="adon-iphone.png"
					description='Unofficial At Dead Of Night port for iPhone. Playable, but has performance problems. Ipa-file must be installed through AltStore or Sideloadly. <a href="https://docs.google.com/spreadsheets/d/1oyyau5VPeA4obLWvlPKQQSB8kgIYfQoX7xH1DXmIr6A/edit?usp=sharing" target="_blank" rel="noreferrer noopener">Devices performance table</a>'
					descriptionRu='Неофициальный порт At Dead Of Night на iPhone. Возможно играть, но имеет проблемы с производительностью. Ipa-файл устанавливается через AltStore или Sideloadly. <a href="https://docs.google.com/spreadsheets/d/1oyyau5VPeA4obLWvlPKQQSB8kgIYfQoX7xH1DXmIr6A/edit?usp=sharing" target="_blank" rel="noreferrer noopener">Таблица производительности по устройствам</a>'
					link={
						'/download?file=' +
						encodeURIComponent('at-dead-of-night-ios-v1.0.0-t.me_adonru.ipa')
					}
					requiresSignIn
				/>
				<Mod
					name="MacOS port"
					image="adon-macos.gif"
					description="At Dead Of Night.app running natively on mac, no Wine or emulators"
					link={'/download?file=' + encodeURIComponent('At Dead Of Night.app')}
					requiresSignIn
				/>
				<Mod
					name="Cheats: radar, items list, wallhack"
					image="cheats.jpeg"
					description={
						"At Dead Of Night cheats. Radar (map with Jimmy's location). Items list. Wallhack (WIP)"
					}
					link="https://github.com/AtDeadOfNight/cheats"
				/>
				<Mod
					name="Russian dubbing"
					image="ru-dub.jpg"
					description="At Dead Of Night with Russian voiceover. Мод для At Dead Of Night с русской озвучкой."
					link="https://adon.hloth.dev"
				/>
				<Mod
					name="At Dead Of Jerma"
					image="adon-jerma.png"
					description="Mod that replaces Jimmy with Jerma985 streamer"
					link="https://github.com/AtDeadOfNight/at-dead-of-jerma-mod"
				/>
				<Mod
					name="At Dead Of deez nuts"
					image="atdeadofdeeznuts.png"
					description="Mod by Max (maxwellheller98@gmail.com)"
					link={'/download?file=' + encodeURIComponent('atdeadofdeeznuts.7z')}
					requiresSignIn
				/>
			</div>
		</section>
	);
}
