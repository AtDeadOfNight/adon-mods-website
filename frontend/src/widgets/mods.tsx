import { Mod } from '@/entity/mod';
import { mods } from '@/shared/mods';

export function Mods() {
	return (
		<section className="bg-neutral-800 flex justify-center p-6 md:p-10">
			<div
				className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 w-full
					max-w-[1200px] gap-6 1280px:gap-8"
			>
				{mods.map((mod, i) => (
					<Mod {...mod} key={i} />
				))}
			</div>
		</section>
	);
}
