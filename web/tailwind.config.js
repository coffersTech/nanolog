/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./index.html", "./js/**/*.js"],
    theme: {
        extend: {
            animation: {
                'slide-in': 'slide-in 0.3s ease-out',
                'pulse': 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
                'spin': 'spin 1.5s linear infinite',
            },
        },
    },
    plugins: [],
}
