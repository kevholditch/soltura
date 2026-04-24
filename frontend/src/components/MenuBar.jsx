import { BookOpen, Dumbbell, MessageCirclePlus, Moon, Sun } from 'lucide-react'

function menuButtonClass(active = false) {
  const base = 'inline-flex items-center gap-2 px-2 sm:px-3 py-1.5 rounded-md font-mono text-sm transition-colors'
  if (active) {
    return `${base} text-amber-800 dark:text-amber-200 bg-amber-50 dark:bg-amber-900/20`
  }
  return `${base} text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800`
}

export default function MenuBar({ activeView, theme, onNewSession, onDrillsStart, onVocabularyOpen, onThemeToggle }) {
  return (
    <nav className="border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 sticky top-0 z-10">
      <div className="max-w-4xl mx-auto px-4 h-14 flex items-center justify-between gap-3">
        <span className="font-fraunces text-xl font-semibold text-amber-800 dark:text-amber-100 tracking-tight">Soltura</span>
        <div className="flex items-center gap-1">
          <button
            onClick={onNewSession}
            aria-label="New chat session"
            className={menuButtonClass(activeView === 'start')}
          >
            <MessageCirclePlus size={16} />
            <span className="hidden sm:inline">New Chat Session</span>
          </button>
          <button
            onClick={onDrillsStart}
            aria-label="Drills"
            className={menuButtonClass(activeView === 'drills')}
          >
            <Dumbbell size={16} />
            <span className="hidden sm:inline">Drills</span>
          </button>
          <button
            onClick={onVocabularyOpen}
            aria-label="Vocabulary"
            className={menuButtonClass(activeView === 'vocab')}
          >
            <BookOpen size={16} />
            <span className="hidden md:inline">Vocabulary</span>
          </button>
          <button
            onClick={onThemeToggle}
            aria-label="Toggle theme"
            className="p-2 rounded-md text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          >
            {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
          </button>
        </div>
      </div>
    </nav>
  )
}
