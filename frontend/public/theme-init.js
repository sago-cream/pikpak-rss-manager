try {
    const preference = localStorage.getItem('theme');
    document.documentElement.dataset.theme =
        preference === 'dark' ||
        (preference !== 'light' &&
            matchMedia('(prefers-color-scheme: dark)').matches)
            ? 'dark'
            : 'light';
} catch {
    document.documentElement.dataset.theme = matchMedia(
        '(prefers-color-scheme: dark)'
    ).matches
        ? 'dark'
        : 'light';
}
