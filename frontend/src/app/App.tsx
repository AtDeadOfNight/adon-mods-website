import { Layout } from '@/app/Layout';
import { RouteRecord } from 'vite-react-ssg';
import { HomePage } from '../pages';
import { PurchasePage } from '@/pages/purchase';
import { NotFoundPage } from '@/pages/not-found';
import { DownloadPage } from '@/pages/download-page';
import { mods } from '@/shared/mods';
import { ErrorSteamAuth } from '@/pages/error-steam-auth';

export const routes: RouteRecord[] = [
	{
		path: '/',
		element: <Layout />,
		children: [
			{ index: true, element: <HomePage /> },
			{
				path: 'purchase-to-download/:file',
				element: <PurchasePage />,
				getStaticPaths: () =>
					mods
						.filter((m) => m.requiresSignIn)
						.map((m) => 'purchase-to-download/' + m.id)
			},
			{
				path: 'download-page/:file',
				element: <DownloadPage />,
				getStaticPaths: () =>
					mods
						.filter((m) => m.requiresSignIn)
						.map((m) => 'download-page/' + m.id)
			},
			{
				path: 'error/game-not-owned',
				element: <ErrorSteamAuth owned={false} />
			},
			{
				path: 'error/game-owned-unknown',
				element: <ErrorSteamAuth owned={'unknown'} />
			},
			{
				path: '*',
				element: <NotFoundPage />
			}
		]
	}
];
