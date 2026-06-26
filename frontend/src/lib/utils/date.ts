const dateFormatter = new Intl.DateTimeFormat('en-US', {
	year: 'numeric',
	month: 'short',
	day: 'numeric'
});

const dateTimeFormatter = new Intl.DateTimeFormat('en-US', {
	year: 'numeric',
	month: 'short',
	day: 'numeric',
	hour: 'numeric',
	minute: '2-digit'
});

const relativeFormatter = new Intl.RelativeTimeFormat('en-US', { numeric: 'auto' });

export function formatDate(date: string | Date): string {
	return dateFormatter.format(new Date(date));
}

export function formatDateTime(date: string | Date): string {
	return dateTimeFormatter.format(new Date(date));
}

export function formatRelative(date: string | Date): string {
	const now = Date.now();
	const diff = new Date(date).getTime() - now;
	const absDiff = Math.abs(diff);

	if (absDiff < 60_000) return relativeFormatter.format(Math.round(diff / 1000), 'second');
	if (absDiff < 3_600_000) return relativeFormatter.format(Math.round(diff / 60_000), 'minute');
	if (absDiff < 86_400_000) return relativeFormatter.format(Math.round(diff / 3_600_000), 'hour');
	return relativeFormatter.format(Math.round(diff / 86_400_000), 'day');
}
