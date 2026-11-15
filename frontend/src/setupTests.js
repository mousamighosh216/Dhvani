import '@testing-library/jest-dom/vitest'

// jsdom has no object URLs.
URL.createObjectURL = () => 'blob:test'
URL.revokeObjectURL = () => {}
