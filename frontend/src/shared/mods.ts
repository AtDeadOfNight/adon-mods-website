export const mods: ({
	name: string;
	nameRu?: string;
	description: string;
	descriptionRu?: string;
	image: string;
} & (
	| {
			id: string;
			requiresSignIn: true;
	  }
	| {
			link: string;
			requiresSignIn: false;
	  }
))[] = [
	{
		name: 'Android port (APK)',
		nameRu: 'Порт на Android (APK)',
		image: 'adon-android.png',
		description:
			'Unofficial At Dead Of Night port for Android. ONLY FOR PHONES WITH AT LEAST 8 GB OF RAM. <a href="https://docs.google.com/spreadsheets/d/1oyyau5VPeA4obLWvlPKQQSB8kgIYfQoX7xH1DXmIr6A/edit?usp=sharing" target="_blank" rel="noreferrer noopener">Check if your device is compatible</a>',
		descriptionRu:
			'Неофициальный порт At Dead Of Night на Android. ТОЛЬКО ДЛЯ УСТРОЙСТВ С 8 ГБ ОПЕРАТИВНОЙ ПАМЯТИ И БОЛЕЕ. <a href="https://docs.google.com/spreadsheets/d/1oyyau5VPeA4obLWvlPKQQSB8kgIYfQoX7xH1DXmIr6A/edit?usp=sharing" target="_blank" rel="noreferrer noopener">Проверьте совместимость вашего устройства</a>',
		id: 'at-dead-of-night-android-v1.0.0-t.me_adonru.apk',
		requiresSignIn: true
	},
	{
		name: 'iPhone port (IPA)',
		nameRu: 'Порт на iPhone (IPA)',
		image: 'adon-iphone.png',
		description:
			'Unofficial At Dead Of Night port for iPhone. Playable, but has performance problems. Ipa-file must be installed through AltStore or Sideloadly. <a href="https://docs.google.com/spreadsheets/d/1oyyau5VPeA4obLWvlPKQQSB8kgIYfQoX7xH1DXmIr6A/edit?usp=sharing" target="_blank" rel="noreferrer noopener">Devices performance table</a>',
		descriptionRu:
			'Неофициальный порт At Dead Of Night на iPhone. Возможно играть, но имеет проблемы с производительностью. Ipa-файл устанавливается через AltStore или Sideloadly. <a href="https://docs.google.com/spreadsheets/d/1oyyau5VPeA4obLWvlPKQQSB8kgIYfQoX7xH1DXmIr6A/edit?usp=sharing" target="_blank" rel="noreferrer noopener">Таблица производительности по устройствам</a>',
		id: 'at-dead-of-night-ios-v1.0.0-t.me_adonru.ipa',
		requiresSignIn: true
	},
	{
		name: 'MacOS port',
		image: 'adon-macos.gif',
		description:
			'At Dead Of Night.app running natively on mac, no Wine or emulators',
		id: 'AtDeadOfNight.app',
		requiresSignIn: true
	},
	{
		name: 'Cheats: radar, items list, wallhack',
		image: 'cheats.jpeg',
		description:
			"At Dead Of Night cheats. Radar (map with Jimmy's location). Items list. Wallhack (WIP)",
		link: 'https://github.com/AtDeadOfNight/cheats',
		requiresSignIn: false
	},
	{
		name: 'Russian dubbing',
		image: 'ru-dub.jpg',
		description:
			'At Dead Of Night with Russian voiceover. Мод для At Dead Of Night с русской озвучкой.',
		link: 'https://adon.hloth.dev',
		requiresSignIn: false
	},
	{
		name: 'At Dead Of Jerma',
		image: 'adon-jerma.png',
		description: 'Mod that replaces Jimmy with Jerma985 streamer',
		link: 'https://github.com/AtDeadOfNight/at-dead-of-jerma-mod',
		requiresSignIn: false
	},
	{
		name: 'At Dead Of deez nuts',
		image: 'atdeadofdeeznuts.png',
		description: 'Mod by Max (maxwellheller98@gmail.com)',
		id: 'atdeadofdeeznuts.7z',
		requiresSignIn: true
	}
];
