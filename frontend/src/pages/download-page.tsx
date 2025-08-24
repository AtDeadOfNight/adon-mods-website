import { useParams, redirect } from 'react-router-dom';
import React from 'react';

export function DownloadPage() {
	const [isRu, setIsRu] = React.useState(false);
	const file = useParams().file;
	if (!file) {
		return redirect('/');
	}
	const downloadUrl = `/download?file=${encodeURIComponent(file)}`;

	React.useEffect(() => {
		setIsRu(navigator.language.startsWith('ru'));
	}, []);

	return (
		<div className="flex items-center justify-center text-white">
			<a href={downloadUrl} className="text-white underline underline-offset-2">
				{isRu ? 'Нажмите, чтобы скачать' : 'Click here to download'}
			</a>
		</div>
	);
}
