import { Hero } from '@/widgets/hero';
import { Mods } from '@/widgets/mods';

export function HomePage() {
	return (
		<main className="w-full min-h-screen">
			<Hero />
			<Mods />
		</main>
	);
}
