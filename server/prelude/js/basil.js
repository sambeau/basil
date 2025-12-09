// Basil Component Enhancements
// Progressive enhancement behaviors for HTML components

// Confirm before submit
document.querySelectorAll( 'form[data-confirm]' ).forEach( f =>
	f.addEventListener( 'submit', e =>
		confirm( f.dataset.confirm ) || e.preventDefault() ) )

// Auto-submit on change
document.querySelectorAll( '[data-autosubmit]' ).forEach( el =>
	el.addEventListener( 'change', () => el.form.submit() ) )

// Character counter
document.querySelectorAll( '[data-counter]' ).forEach( ta => {
	const counter = document.getElementById( ta.dataset.counter )
	const max = ta.maxLength
	const update = () => counter.textContent = `${ta.value.length} / ${max}`
	ta.addEventListener( 'input', update )
	update()
} )

// Toggle visibility
document.querySelectorAll( '[data-toggle]' ).forEach( btn => {
	const target = document.querySelector( btn.dataset.toggle )
	btn.setAttribute( 'aria-controls', target.id )
	btn.setAttribute( 'aria-expanded', !target.hidden )
	btn.addEventListener( 'click', () => {
		target.hidden = !target.hidden
		btn.setAttribute( 'aria-expanded', !target.hidden )
	} )
} )

// Copy to clipboard
document.querySelectorAll( '[data-copy]' ).forEach( btn => {
	const originalText = btn.textContent
	btn.addEventListener( 'click', async () => {
		try {
			const text = document.querySelector( btn.dataset.copy ).textContent
			await navigator.clipboard.writeText( text )
			btn.textContent = 'Copied!'
		} catch ( e ) {
			btn.textContent = 'Failed'
		}
		setTimeout( () => btn.textContent = originalText, 2000 )
	} )
} )

// Disable submit button on submit
document.querySelectorAll( 'form' ).forEach( f =>
	f.addEventListener( 'submit', () =>
		f.querySelectorAll( '[type=submit]' ).forEach( b => b.disabled = true ) ) )

// Auto-resize textarea (CSS fallback)
if ( !CSS.supports( 'field-sizing', 'content' ) ) {
	document.querySelectorAll( '[data-autoresize]' ).forEach( ta => {
		const resize = () => { ta.style.height = 'auto'; ta.style.height = ta.scrollHeight + 'px' }
		ta.addEventListener( 'input', resize )
		resize()
	} )
}

// Focus first invalid field
const firstError = document.querySelector( '[aria-invalid="true"]' )
if ( firstError ) firstError.focus()
