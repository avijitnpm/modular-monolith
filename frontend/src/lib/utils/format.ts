const usdFormatter = new Intl.NumberFormat('en-US', {
	style: 'currency',
	currency: 'USD',
});

const usdCompactFormatter = new Intl.NumberFormat('en-US', {
	style: 'currency',
	currency: 'USD',
	notation: 'compact',
});

export function formatCurrency(cents: number): string {
	return usdFormatter.format(cents / 100);
}

export function formatCurrencyCompact(cents: number): string {
	return usdCompactFormatter.format(cents / 100);
}
