import React from 'react';

export function ErrorSteamAuth({ owned }: { owned: false | 'unknown' }) {
	const [isRu, setIsRu] = React.useState(false);

	React.useEffect(() => {
		setIsRu(navigator.language.startsWith('ru'));
	}, []);

	return (
		<main
			className="flex flex-col items-center justify-center px-16 text-center"
		>
			<section className="w-full max-w-[800px] flex flex-col gap-6 items-center">
				<p className="text-xl font-bold text-red-600">
					{isRu ? 'Ошибка: ' : 'Error: '}
					{owned === 'unknown'
						? isRu
							? 'Мы не смогли получить вашу библиотеку игр в Steam. Возможно, у вас скрыт профиль или библиотека игр. OpenID запрещает доступ к библиотеке Steam. Вы должны открыть ваш профиль, чтобы доказать, что у вас есть копия игры. После этого авторизуйтесь на этом сайте снова.'
							: "We couldn't access your Steam library. Most likely, that's because your profile is hidden. OpenID restricts access to private Steam profiles. You must open your Steam profile to prove that you own At Dead Of Night. Then please click the Sign In button again. "
						: isRu
							? 'У вас должна быть куплена At Dead Of Night в Steam для скачивания.'
							: 'You must have an At Dead Of Night copy purchased in Steam to download this mod.'}
				</p>
				<a href="/" className="mt-5">
					{isRu ? 'Вернуться на главную страницу' : 'Return to the main page'}
				</a>
			</section>
		</main>
	);
}
