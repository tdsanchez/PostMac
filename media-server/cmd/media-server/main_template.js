const filePath = '{{.File.RelPath}}';
const isText = {{isTextFile .File.Name}};
const isConvertible = {{isConvertibleFile .File.Name}};
const currentTag = '{{.Tag}}';
const totalFiles = {{.Total}};
const prevFilePath = '{{.PrevFile.RelPath}}';
const nextFilePath = '{{.NextFile.RelPath}}';

// Try to load file paths from localStorage cache (populated by gallery view)
// If cache unavailable, will fetch on-demand from /api/filelist when random mode used
let allFilePaths = [];
try {
	const cacheKey = 'fileList_' + currentTag;
	const cached = localStorage.getItem(cacheKey);
	if (cached) {
		allFilePaths = JSON.parse(cached);
	}
} catch (e) {
	console.warn('Failed to load cached file list:', e);
}

let allTags = [];
let selectedIndex = -1;
let contextMenuTag = null;

let slideshowActive = false;
let slideshowInterval = null;
let slideshowDelay = 3000;
let randomMode = false;

const starRatings = {
	'1': '1-★',
	'2': '2-★★',
	'3': '3-★★★',
	'4': '4-★★★★',
	'5': '5-★★★★★'
};

const urlParams = new URLSearchParams(window.location.search);
randomMode = urlParams.get('random') === 'true';

if (urlParams.get('slideshow') === 'true') {
	slideshowActive = true;
	slideshowDelay = parseInt(urlParams.get('delay')) || 3000;
	document.getElementById('slideshow-indicator').classList.add('active');
	updateSlideshowIndicator();
	startSlideshow();
}

if (isText) {
	fetch('/file/' + encodeURIComponent(filePath))
		.then(r => r.text())
		.then(text => {
			document.getElementById('text-content').textContent = text;
		})
		.catch(err => {
			document.getElementById('text-content').textContent = 'Error loading file: ' + err.message;
		});
}

function buildURL(filepath) {
	const params = new URLSearchParams();
	params.set('file', filepath);
	if (slideshowActive) {
		params.set('slideshow', 'true');
		params.set('delay', slideshowDelay);
	}
	if (randomMode) {
		params.set('random', 'true');
	}
	return '/view/' + encodeURIComponent(currentTag) + '?' + params.toString();
}

function navigatePrev() {
	// Left arrow always goes back in browser history
	window.history.back();
}

function navigateNext() {
	if (randomMode) {
		navigateRandom();
	} else {
		window.location.href = buildURL(nextFilePath);
	}
}

async function navigateRandom() {
	// If file list not in cache, fetch from API
	if (allFilePaths.length === 0) {
		try {
			const response = await fetch('/api/filelist?category=' + encodeURIComponent(currentTag));
			if (!response.ok) {
				throw new Error('Failed to fetch file list');
			}
			allFilePaths = await response.json();

			// Cache the fetched list for future use
			try {
				const cacheKey = 'fileList_' + currentTag;
				localStorage.setItem(cacheKey, JSON.stringify(allFilePaths));
			} catch (e) {
				console.warn('Failed to cache file list:', e);
			}
		} catch (err) {
			console.error('Error fetching file list:', err);
			// Fallback to next file if API fetch fails
			window.location.href = buildURL(nextFilePath);
			return;
		}
	}

	if (allFilePaths.length === 0) {
		// Still empty after fetch - fallback to next
		window.location.href = buildURL(nextFilePath);
		return;
	}

	// Pick a random file path, ensuring it's different from current file
	let randomPath;
	if (allFilePaths.length === 1) {
		randomPath = allFilePaths[0];
	} else {
		do {
			randomPath = allFilePaths[Math.floor(Math.random() * allFilePaths.length)];
		} while (randomPath === filePath && allFilePaths.length > 1);
	}

	window.location.href = buildURL(randomPath);
}

function quickLookPreview() {
	fetch('/api/quicklook', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ filePath: filePath })
	})
	.then(r => r.json())
	.then(data => {
		if (data.success) {
			showNotification('👁 QuickLook opened');
		}
	})
	.catch(err => {
		console.error('Error launching QuickLook:', err);
		showNotification('❌ Failed to open QuickLook');
	});
}

fetch('/api/alltags')
	.then(r => r.json())
	.then(tags => { allTags = tags; });

fetch('/api/metadata/' + encodeURIComponent(filePath))
	.then(r => r.json())
	.then(data => {
		const metadataEl = document.getElementById('metadata-inline');
		const parts = [];

		if (data.width && data.height) {
			parts.push(data.width + '×' + data.height);
		}
		if (data.fileSize) {
			const sizeInMB = data.fileSize / (1024*1024);
			const size = (sizeInMB >= 1)
				? sizeInMB.toFixed(1) + 'MB'
				: (data.fileSize / 1024).toFixed(1) + 'KB';
			parts.push(size);
		}
		if (data.created) {
			parts.push(new Date(data.created).toLocaleDateString());
		}

		metadataEl.textContent = parts.length > 0 ? '[' + parts.join(' | ') + ']' : '[No metadata]';
	})
	.catch(err => {
		console.error('Metadata fetch error:', err);
		document.getElementById('metadata-inline').textContent = '[Error loading metadata]';
	});

function showNotification(message) {
	const notif = document.getElementById('notification');
	notif.textContent = message;
	notif.classList.add('show');
	setTimeout(() => notif.classList.remove('show'), 2000);
}

function toggleSlideshow() {
	slideshowActive = !slideshowActive;
	const indicator = document.getElementById('slideshow-indicator');

	if (slideshowActive) {
		indicator.classList.add('active');
		updateSlideshowIndicator();
		startSlideshow();
		showNotification('▶️ Slideshow started');
	} else {
		indicator.classList.remove('active');
		stopSlideshow();
		showNotification('⏹️ Slideshow stopped');
	}
}

function startSlideshow() {
	if (slideshowInterval) {
		clearInterval(slideshowInterval);
	}
	slideshowInterval = setInterval(() => {
		if (randomMode) {
			navigateRandom();
		} else {
			navigateNext();
		}
	}, slideshowDelay);
}

function stopSlideshow() {
	if (slideshowInterval) {
		clearInterval(slideshowInterval);
		slideshowInterval = null;
	}
}

function adjustSlideshowTiming(faster) {
	if (faster) {
		slideshowDelay = Math.max(250, slideshowDelay - 50);
	} else {
		slideshowDelay = Math.min(25000, slideshowDelay + 50);
	}
	updateSlideshowIndicator();
	if (slideshowActive) {
		startSlideshow();
	}
	showNotification('⏱️ Timing: ' + (slideshowDelay / 1000) + 's');
}

function updateSlideshowIndicator() {
	const indicator = document.getElementById('slideshow-indicator');
	const mode = randomMode ? 'RANDOM' : 'SLIDESHOW';
	indicator.textContent = '▶️ ' + mode + ': ' + (slideshowDelay / 1000) + 's';
}

function addTag(tagName) {
	fetch('/api/addtag', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ filePath: filePath, tag: tagName })
	})
	.then(r => r.json())
	.then(data => {
		if (data.success) {
			showNotification('✅ Tag added: ' + tagName);
			updateTagsDisplay(data.tags);
		}
	})
	.catch(err => console.error('Error adding tag:', err));
}

function removeTag(tagName) {
	fetch('/api/removetag', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ filePath: filePath, tag: tagName })
	})
	.then(r => r.json())
	.then(data => {
		if (data.success) {
			showNotification('✅ Tag removed: ' + tagName);
			updateTagsDisplay(data.tags);
		}
	})
	.catch(err => console.error('Error removing tag:', err));
}

function updateTagsDisplay(tags) {
	const container = document.getElementById('tags-container');
	container.innerHTML = '';
	tags.forEach(tag => {
		const a = document.createElement('a');
		a.href = '/tag/' + encodeURIComponent(tag);
		a.className = 'tag';
		a.setAttribute('data-tag', tag);
		a.textContent = tag;
		container.appendChild(a);
	});
	attachTagListeners();
}

function attachTagListeners() {
	document.querySelectorAll('.tag').forEach(tag => {
		tag.addEventListener('contextmenu', function(e) {
			e.preventDefault();
			contextMenuTag = this.getAttribute('data-tag');
			const menu = document.getElementById('context-menu');
			menu.style.display = 'block';
			menu.style.left = e.pageX + 'px';
			menu.style.top = e.pageY + 'px';
		});

		// Regular clicks now navigate to tag gallery (matching gallery view behavior)
		// Right-click context menu still available for delete/goto options
	});
}

document.addEventListener('click', function() {
	document.getElementById('context-menu').style.display = 'none';
});

document.getElementById('menu-delete').addEventListener('click', function() {
	if (contextMenuTag) {
		removeTag(contextMenuTag);
	}
	document.getElementById('context-menu').style.display = 'none';
});

document.getElementById('menu-goto').addEventListener('click', function() {
	if (contextMenuTag) {
		window.location.href = '/tag/' + encodeURIComponent(contextMenuTag);
	}
});

attachTagListeners();

const tagInput = document.getElementById('tag-input');
const tagInputContainer = document.getElementById('tag-input-container');
const autocomplete = document.getElementById('autocomplete');

// Helper function to exit tag editing mode
function exitTagMode() {
	if (tagInputContainer.classList.contains('active')) {
		tagInput.value = '';
		autocomplete.innerHTML = '';
		selectedIndex = -1;
		tagInputContainer.classList.remove('active');
	}
}

tagInput.addEventListener('input', function() {
	const value = this.value.toLowerCase();
	if (!value) {
		autocomplete.innerHTML = '';
		return;
	}

	const matches = allTags.filter(t => t.toLowerCase().includes(value));
	autocomplete.innerHTML = '';
	selectedIndex = -1;

	matches.forEach((tag, idx) => {
		const div = document.createElement('div');
		div.className = 'autocomplete-item';
		div.textContent = tag;
		div.addEventListener('click', () => {
			addTag(tag);
			tagInput.value = '';
			autocomplete.innerHTML = '';
		});
		autocomplete.appendChild(div);
	});
});

tagInput.addEventListener('keydown', function(e) {
	const items = autocomplete.querySelectorAll('.autocomplete-item');

	if (e.key === 'ArrowDown') {
		e.preventDefault();
		selectedIndex = Math.min(selectedIndex + 1, items.length - 1);
		updateSelection(items);
	} else if (e.key === 'ArrowUp') {
		e.preventDefault();
		selectedIndex = Math.max(selectedIndex - 1, -1);
		updateSelection(items);
	} else if (e.key === 'Enter') {
		e.preventDefault();
		if (selectedIndex !== -1 && items[selectedIndex]) {
			addTag(items[selectedIndex].textContent);
		} else if (this.value.trim()) {
			addTag(this.value.trim());
		}
		exitTagMode();
	} else if (e.key === 'Escape') {
		e.preventDefault();
		exitTagMode();
	}
});

function updateSelection(items) {
	items.forEach((item, idx) => {
		item.classList.toggle('selected', idx === selectedIndex);
	});
	if (selectedIndex !== -1 && items[selectedIndex]) {
		items[selectedIndex].scrollIntoView({ block: 'nearest' });
	}
}

document.addEventListener('keydown', function(e) {
	// Don't intercept keystrokes while editing tags or comments
	if (document.activeElement === tagInput) return;
	if (document.activeElement && document.activeElement.id === 'comment-edit') return;

	// Don't intercept browser shortcuts (Cmd+L, Cmd+Plus, Cmd+Minus, etc.)
	if (e.metaKey || e.ctrlKey) return;

	if (e.key === 'ArrowLeft') {
		exitTagMode();
		navigatePrev();
	} else if (e.key === 'ArrowRight') {
		exitTagMode();
		navigateNext();
	} else if (e.key === 'Escape') {
		exitTagMode();
		if (slideshowActive) {
			toggleSlideshow();
		} else {
			window.location.href = '/tag/' + encodeURIComponent(currentTag);
		}
	} else if (e.key === 's' || e.key === 'S') {
		e.preventDefault();
		toggleSlideshow();
	} else if (e.key === 'r' || e.key === 'R') {
		e.preventDefault();
		randomMode = !randomMode;
		updateSlideshowIndicator();
		showNotification(randomMode ? '🎲 Random mode ON' : '⏭️ Sequential mode ON');
	} else if (e.key === 'x' || e.key === 'X') {
		e.preventDefault();
		showDeleteModal();
	} else if (e.key === '+' || e.key === '=') {
		e.preventDefault();
		adjustSlideshowTiming(true);
	} else if (e.key === '-' || e.key === '_') {
		e.preventDefault();
		adjustSlideshowTiming(false);
	} else if (e.key === 't' || e.key === 'T') {
		e.preventDefault();
		tagInputContainer.classList.add('active');
		tagInput.focus();
	} else if (e.key === 'c' || e.key === 'C') {
		e.preventDefault();
		enableCommentEditing();
	} else if (e.key === 'q' || e.key === 'Q') {
		e.preventDefault();
		quickLookPreview();
	} else if (e.key === 'l' || e.key === 'L') {
		e.preventDefault();
		addTag('❤️');
	} else if (starRatings[e.key]) {
		e.preventDefault();
		addTag(starRatings[e.key]);
	}
});

window.addEventListener('beforeunload', function() {
	stopSlideshow();
});

// ============================================================================
// COMMENT EDITING FUNCTIONS
// ============================================================================

let isEditingComment = false;

function enableCommentEditing() {
	const display = document.getElementById('comment-display');
	const textarea = document.getElementById('comment-edit');
	const container = display.parentElement;

	// Get current comment
	const currentComment = display.textContent;

	// Show textarea, hide display
	display.style.display = 'none';
	textarea.style.display = 'block';
	textarea.value = currentComment === 'Click to add comment...' ? '' : currentComment;
	textarea.focus();

	isEditingComment = true;

	// Save on blur
	const blurHandler = () => {
		saveComment();
		textarea.removeEventListener('blur', blurHandler);
	};
	textarea.addEventListener('blur', blurHandler);

	// Save on Ctrl+Enter or Cmd+Enter, Cancel on Escape
	const keyHandler = (e) => {
		if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
			e.preventDefault();
			saveComment();
			textarea.removeEventListener('keydown', keyHandler);
		}
		if (e.key === 'Escape') {
			e.preventDefault();
			cancelCommentEdit();
			textarea.removeEventListener('keydown', keyHandler);
		}
	};
	textarea.addEventListener('keydown', keyHandler);
}

function cancelCommentEdit() {
	const display = document.getElementById('comment-display');
	const textarea = document.getElementById('comment-edit');

	textarea.style.display = 'none';
	display.style.display = 'block';
	isEditingComment = false;
}

async function saveComment() {
	const display = document.getElementById('comment-display');
	const textarea = document.getElementById('comment-edit');
	const container = display.parentElement;
	const newComment = textarea.value.trim();

	// Debug logging
	console.log('=== saveComment DEBUG ===');
	console.log('filePath:', filePath);
	console.log('newComment:', newComment);

	// Show saving state
	container.classList.add('comment-saving');

	try {
		const payload = {
			filepath: filePath,
			comment: newComment
		};
		console.log('Sending payload:', JSON.stringify(payload));

		const response = await fetch('/api/comment', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify(payload)
		});

		console.log('Response status:', response.status);
		console.log('Response ok:', response.ok);

		if (!response.ok) {
			throw new Error('Failed to update comment');
		}

		// Update UI
		display.textContent = newComment;

		// Update empty state
		if (newComment === '') {
			display.classList.add('empty');
		} else {
			display.classList.remove('empty');
		}

		showNotification('✅ Comment saved');
		console.log('Comment saved successfully');

	} catch (error) {
		console.error('Error saving comment:', error);
		showNotification('❌ Failed to save comment');
	} finally {
		// Hide textarea, show display
		container.classList.remove('comment-saving');
		textarea.style.display = 'none';
		display.style.display = 'block';
		isEditingComment = false;
	}
}

// Click to edit comment
document.getElementById('comment-display').addEventListener('click', function() {
	enableCommentEditing();
});

// ============================================================================
// SHUTDOWN MODAL AND FUNCTIONALITY
// ============================================================================

function showShutdownModal() {
	document.getElementById('shutdownModal').classList.add('show');
}

function hideShutdownModal() {
	document.getElementById('shutdownModal').classList.remove('show');
}

async function confirmShutdown() {
	try {
		const response = await fetch('/api/shutdown', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			}
		});

		if (response.ok) {
			// Update modal to show shutdown message
			const modal = document.querySelector('.modal');
			modal.innerHTML = `
				<div class="modal-title">👋 Shutting Down</div>
				<div class="modal-message">Media server is shutting down...</div>
			`;

			// Give server time to shutdown, then close window
			setTimeout(() => {
				window.close();
			}, 2000);
		} else {
			alert('Failed to shutdown server. Please try again.');
			hideShutdownModal();
		}
	} catch (error) {
		console.error('Error shutting down server:', error);
		// Server already shutdown - update UI
		const modal = document.querySelector('.modal');
		modal.innerHTML = `
			<div class="modal-title">✅ Server Stopped</div>
			<div class="modal-message">Media server has been shut down.</div>
		`;
		setTimeout(() => {
			window.close();
		}, 1500);
	}
}

// Close modal when clicking outside of it
document.getElementById('shutdownModal').addEventListener('click', function(e) {
	if (e.target === this) {
		hideShutdownModal();
	}
});

// Update ESC key handler to also close shutdown modal
document.addEventListener('keydown', function(e) {
	if (e.key === 'Escape') {
		hideShutdownModal();
		hideDeleteModal();
	}
});

// ============================================================================
// DELETE FILE MODAL AND FUNCTIONALITY
// ============================================================================

function showDeleteModal() {
	document.getElementById('deleteModal').classList.add('show');
}

function hideDeleteModal() {
	document.getElementById('deleteModal').classList.remove('show');
}

async function confirmDelete() {
	try {
		const response = await fetch('/api/deletefile', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({ filePath: filePath })
		});

		if (response.ok) {
			const data = await response.json();

			// Update modal to show success message
			const modal = document.querySelector('#deleteModal .modal');
			modal.innerHTML = `
				<div class="modal-title">✅ File Deleted</div>
				<div class="modal-message">File moved to Trash. Navigating to next file...</div>
			`;

			// Navigate to next file after short delay
			setTimeout(() => {
				hideDeleteModal();
				if (randomMode) {
					navigateRandom();
				} else {
					window.location.href = buildURL(nextFilePath);
				}
			}, 1000);
		} else {
			const data = await response.json();
			alert('Failed to delete file: ' + (data.error || 'Unknown error'));
			hideDeleteModal();
		}
	} catch (error) {
		console.error('Error deleting file:', error);
		alert('Error deleting file: ' + error.message);
		hideDeleteModal();
	}
}

// Close delete modal when clicking outside of it
document.getElementById('deleteModal').addEventListener('click', function(e) {
	if (e.target === this) {
		hideDeleteModal();
	}
});
