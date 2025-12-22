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

// =============================================================================
// Time Components - Timezone Localization
// =============================================================================

// Helper: Get Intl.DateTimeFormat options from format string
function getDateTimeOptions ( format, weekday, showZone ) {
	const base = {}

	switch ( format ) {
		case 'short':
			base.dateStyle = 'short'
			base.timeStyle = 'short'
			break
		case 'full':
			base.dateStyle = 'full'
			base.timeStyle = 'long'
			break
		case 'date':
			base.dateStyle = 'long'
			break
		case 'time':
			base.timeStyle = 'short'
			break
		case 'long':
		default:
			base.dateStyle = 'long'
			base.timeStyle = 'short'
			break
	}

	if ( weekday ) {
		base.weekday = weekday // 'short' or 'long'
	}

	if ( showZone && base.timeStyle ) {
		base.timeZoneName = 'short'
	}

	return base
}

// <local-time> - Convert UTC datetime to user's local timezone
class LocalTimeElement extends HTMLElement {
	connectedCallback () {
		this.enhance()
	}

	enhance () {
		const datetime = this.getAttribute( 'datetime' )
		if ( !datetime ) return

		try {
			const date = new Date( datetime )
			if ( isNaN( date ) ) return

			const format = this.getAttribute( 'format' ) || 'long'
			const weekday = this.getAttribute( 'weekday' )
			const showZone = this.hasAttribute( 'show-zone' )

			const options = getDateTimeOptions( format, weekday, showZone )
			this.textContent = new Intl.DateTimeFormat( navigator.language, options ).format( date )
		} catch ( e ) {
			// Keep server-rendered fallback on error
		}
	}
}

if ( !customElements.get( 'local-time' ) ) {
	customElements.define( 'local-time', LocalTimeElement )
}

// <time-range> - Smart display of datetime spans with local timezone
class TimeRangeElement extends HTMLElement {
	connectedCallback () {
		this.enhance()
	}

	enhance () {
		const startAttr = this.getAttribute( 'start' )
		const endAttr = this.getAttribute( 'end' )
		if ( !startAttr || !endAttr ) return

		try {
			const start = new Date( startAttr )
			const end = new Date( endAttr )
			if ( isNaN( start ) || isNaN( end ) ) return

			const format = this.getAttribute( 'format' ) || 'long'
			const separator = this.getAttribute( 'separator' ) || ' – '
			const lang = navigator.language

			// Check if same day, month, year
			const sameDay = start.toDateString() === end.toDateString()
			const sameMonth = start.getMonth() === end.getMonth() && start.getFullYear() === end.getFullYear()
			const sameYear = start.getFullYear() === end.getFullYear()

			let text

			if ( sameDay ) {
				// Same day: "December 25, 2024, 9:00 AM – 11:00 AM"
				const dateFormatter = new Intl.DateTimeFormat( lang, { dateStyle: 'long' } )
				const timeFormatter = new Intl.DateTimeFormat( lang, { timeStyle: 'short' } )
				text = `${dateFormatter.format( start )}, ${timeFormatter.format( start )}${separator}${timeFormatter.format( end )}`
			} else if ( sameMonth ) {
				// Same month: "December 25 – 27, 2024"
				const dayFormatter = new Intl.DateTimeFormat( lang, { day: 'numeric' } )
				const monthYearFormatter = new Intl.DateTimeFormat( lang, { month: 'long', year: 'numeric' } )
				text = `${monthYearFormatter.format( start ).replace( /\d+/, '' ).trim()} ${dayFormatter.format( start )}${separator}${dayFormatter.format( end )}, ${start.getFullYear()}`
			} else if ( sameYear ) {
				// Same year: "December 25 – January 3, 2024"
				const monthDayFormatter = new Intl.DateTimeFormat( lang, { month: 'long', day: 'numeric' } )
				text = `${monthDayFormatter.format( start )}${separator}${monthDayFormatter.format( end )}, ${start.getFullYear()}`
			} else {
				// Different years: full dates
				const fullFormatter = new Intl.DateTimeFormat( lang, { dateStyle: 'long' } )
				text = `${fullFormatter.format( start )}${separator}${fullFormatter.format( end )}`
			}

			this.textContent = text
		} catch ( e ) {
			// Keep server-rendered fallback on error
		}
	}
}

if ( !customElements.get( 'time-range' ) ) {
	customElements.define( 'time-range', TimeRangeElement )
}

// <relative-time> - Human-readable relative time with optional auto-refresh
class RelativeTimeElement extends HTMLElement {
	connectedCallback () {
		this.update()

		if ( this.hasAttribute( 'live' ) ) {
			this.startAutoRefresh()
		}
	}

	disconnectedCallback () {
		this.stopAutoRefresh()
	}

	update () {
		const datetime = this.getAttribute( 'datetime' )
		if ( !datetime ) return

		try {
			const date = new Date( datetime )
			if ( isNaN( date ) ) return

			const now = new Date()
			const diff = now - date // milliseconds
			const absDiff = Math.abs( diff )
			const isPast = diff > 0

			// Check threshold - switch to absolute date after duration
			const threshold = this.getAttribute( 'threshold' )
			if ( threshold ) {
				const thresholdMs = this.parseThreshold( threshold )
				if ( thresholdMs && absDiff > thresholdMs ) {
					// Show absolute date instead
					const format = this.getAttribute( 'format' ) || 'long'
					const options = getDateTimeOptions( format, null, false )
					// Only use date options for threshold display
					delete options.timeStyle
					this.textContent = new Intl.DateTimeFormat( navigator.language, options ).format( date )
					return
				}
			}

			// Calculate relative time
			const rtf = new Intl.RelativeTimeFormat( navigator.language, { numeric: 'auto' } )

			const seconds = Math.floor( absDiff / 1000 )
			const minutes = Math.floor( seconds / 60 )
			const hours = Math.floor( minutes / 60 )
			const days = Math.floor( hours / 24 )
			const weeks = Math.floor( days / 7 )
			const months = Math.floor( days / 30 )
			const years = Math.floor( days / 365 )

			let text
			if ( seconds < 60 ) {
				text = rtf.format( isPast ? -seconds : seconds, 'second' )
				// Most RTFs say "in 0 seconds" for 0, let's use "just now" 
				if ( seconds < 10 ) text = isPast ? 'just now' : 'momentarily'
			} else if ( minutes < 60 ) {
				text = rtf.format( isPast ? -minutes : minutes, 'minute' )
			} else if ( hours < 24 ) {
				text = rtf.format( isPast ? -hours : hours, 'hour' )
			} else if ( days < 7 ) {
				text = rtf.format( isPast ? -days : days, 'day' )
			} else if ( weeks < 5 ) {
				text = rtf.format( isPast ? -weeks : weeks, 'week' )
			} else if ( months < 12 ) {
				text = rtf.format( isPast ? -months : months, 'month' )
			} else {
				text = rtf.format( isPast ? -years : years, 'year' )
			}

			// Handle announcements for screen readers at key intervals
			const shouldAnnounce = this.hasAttribute( 'announce' ) && this.lastText && this.lastText !== text
			if ( shouldAnnounce ) {
				// Only announce at meaningful intervals: 1hr, 30min, 10min, 5min, 1min
				const keyMinutes = [ 60, 30, 10, 5, 1 ]
				const isKeyInterval = keyMinutes.includes( minutes ) && seconds % 60 < 5
				if ( isKeyInterval ) {
					this.setAttribute( 'aria-live', 'polite' )
				} else {
					this.setAttribute( 'aria-live', 'off' )
				}
			}

			this.lastText = text
			this.textContent = text
		} catch ( e ) {
			// Keep server-rendered fallback on error
		}
	}

	parseThreshold ( threshold ) {
		// Parse duration strings like "7d", "1w", "30d", "1h"
		const match = threshold.match( /^(\d+)(s|m|h|d|w|mo|y)$/ )
		if ( !match ) return null

		const value = parseInt( match[ 1 ], 10 )
		const unit = match[ 2 ]

		const multipliers = {
			's': 1000,
			'm': 60 * 1000,
			'h': 60 * 60 * 1000,
			'd': 24 * 60 * 60 * 1000,
			'w': 7 * 24 * 60 * 60 * 1000,
			'mo': 30 * 24 * 60 * 60 * 1000,
			'y': 365 * 24 * 60 * 60 * 1000
		}

		return value * ( multipliers[ unit ] || 0 )
	}

	startAutoRefresh () {
		// Determine refresh interval based on how far away the datetime is
		const datetime = this.getAttribute( 'datetime' )
		if ( !datetime ) return

		const date = new Date( datetime )
		const now = new Date()
		const absDiff = Math.abs( now - date )

		// Refresh more frequently when close to the target time
		// < 1 minute: every second
		// < 1 hour: every 30 seconds  
		// otherwise: every minute
		let interval
		if ( absDiff < 60 * 1000 ) {
			interval = 1000
		} else if ( absDiff < 60 * 60 * 1000 ) {
			interval = 30 * 1000
		} else {
			interval = 60 * 1000
		}

		this._refreshInterval = setInterval( () => {
			// Pause when tab is hidden
			if ( document.hidden ) return

			// Stop if element removed from DOM
			if ( !document.body.contains( this ) ) {
				this.stopAutoRefresh()
				return
			}

			this.update()
		}, interval )
	}

	stopAutoRefresh () {
		if ( this._refreshInterval ) {
			clearInterval( this._refreshInterval )
			this._refreshInterval = null
		}
	}
}

if ( !customElements.get( 'relative-time' ) ) {
	customElements.define( 'relative-time', RelativeTimeElement )
}
