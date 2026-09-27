import { mount } from 'svelte'
import '@harvey-withington/folder-templates-ui/styles.css'
import './app.css'
import App from './App.svelte'

const target = document.getElementById('app')
if (!target) throw new Error('#app missing')

export default mount(App, { target })
