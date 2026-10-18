export function Hero() {
	const isRu = navigator.language?.startsWith('ru') ?? false;
	return (
		<section
			className="flex flex-col gap-6 md:gap-4 items-center justify-center w-full
				h-screen text-center relative px-8"
		>
			<h1 className="m-0 font-bold">At Dead Of Night mods</h1>
			{isRu ? (
				<p>
					Добро пожаловать на сайт модов ADON! Вы можете добавить свои моды,
					отправив письмо на <a href="mailto:adon@hloth.dev">adon@hloth.dev</a>
				</p>
			) : (
				<p>
					Welcome to ADON modding community. Please submit your mods by sending
					email to <a href="mailto:adon@hloth.dev">adon@hloth.dev</a>
				</p>
			)}
			<div className="absolute bottom-8 left-1/2 -translate-x-1/2">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="3em"
					viewBox="0 0 24 24"
					className="h-auto aspect-square opacity-50 animate-bounce"
				>
					<path
						fill="currentColor"
						d="M11 4v12.175l-5.6-5.6L4 12l8 8l8-8l-1.4-1.425l-5.6 5.6V4z"
					></path>
				</svg>
			</div>
		</section>
	);
}
