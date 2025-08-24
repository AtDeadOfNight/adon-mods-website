import React from 'react';
import { toast } from 'sonner';

export function PurchasePage() {
	let signIn = '/auth/steam';
	const [isRu, setIsRu] = React.useState(false);

	React.useEffect(() => {
		setIsRu(navigator.language.startsWith('ru'));
	}, []);

	React.useEffect(() => {
		const file = new URLSearchParams(window.location.search).get('file');
		if (file) {
			signIn += '?file=' + encodeURIComponent(file);
		}

		if (
			new URLSearchParams(window.location.search).get('success') === 'false'
		) {
			setTimeout(() => {
				toast.error(
					isRu
						? 'У вас должна быть куплена At Dead Of Night в Steam для скачивания.'
						: 'You must own At Dead Of Night in Steam to download this mod.'
				);
			}, 0);
		} else if (
			new URLSearchParams(window.location.search).get('success') === 'unknown'
		) {
			setTimeout(() => {
				toast.warning(
					isRu
						? 'Мы не смогли получить вашу библиотеку игр в Steam. Возможно, у вас скрыт профиль или библиотека игр. OpenID запрещает доступ к библиотеке Steam. Вы должны открыть ваш профиль, чтобы доказать, что у вас есть копия игры. После этого авторизуйтесь на этом сайте снова.'
						: "We couldn't access your Steam library. Most likely, that's because your profile is hidden. OpenID restricts access to private Steam profiles. You must open your Steam profile to prove that you own At Dead Of Night. Then please click the Sign In button again. ",
					{ duration: 10000 }
				);
			}, 0);
		}
	}, []);

	return (
		<main
			className="flex flex-col items-center justify-center px-16 text-center"
		>
			<section className="w-full max-w-[600px] flex flex-col gap-6 items-center">
				<p className="text-xl font-bold">
					{isRu ? (
						<>
							Для скачивания этого мода, вам нужно купить{' '}
							<a href="https://store.steampowered.com/app/1450830">
								At Dead Of Night в Steam
							</a>
							.
						</>
					) : (
						<>
							To download this mod, you must own{' '}
							<a href="https://store.steampowered.com/app/1450830">
								At Dead Of Night in Steam
							</a>
							.
						</>
					)}
				</p>
				<p className="font-semibold">
					{isRu
						? 'Используйте кнопку ниже, чтобы авторизоваться с Steam для подтверждения владения лицензионной копией игры'
						: 'Please use button below to authorize with your steam account to prove you have a purchased copy of the game.'}
				</p>
				<p className="text-sm text-neutral-600">
					{isRu
						? 'Не волнуйтесь, вам не придется вводить свои учетные данные Steam. Мы используем OpenID аутентификацию. Мы никогда не получаем конфиденциальную информацию из вашего аккаунта, как и указано на странице авторизации Steam. Мы также не храним никакой информации о вас. Мы выполняем только одноразовую проверку, есть ли у вас At Dead Of Night в вашей библиотеке, и вам не придется авторизовываться снова, чтобы скачать другие моды, требующие покупки игры в Steam.'
						: "Don't worry, you won't have to enter your Steam credentials. We use OpenID authentification. We never receive any private info from your account, as stated on Steam authorization page. We also do not store any information about you. We only perform one-time check if you have At Dead Of Night in your library and you won't have to authorize again in order to download other mods that require purchase in Steam."}
				</p>
				<a href={signIn} className="mt-5">
					<img
						src="https://community.akamai.steamstatic.com/public/images/signinthroughsteam/sits_01.png"
						alt="Sign In Button"
					/>
				</a>
			</section>
		</main>
	);
}
