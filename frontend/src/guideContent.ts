// The Guide's words, held apart from the dialog that draws them (ported from
// SymDiary's guideContent.ts). It names each control with its real picture,
// then states the rules the window cannot say for itself (FR-071).

import addIcon from './assets/icons/add-website.png'
import editIcon from './assets/icons/edit-website.png'
import deleteIcon from './assets/icons/delete-website.png'
import refreshIcon from './assets/icons/refresh.png'
import settingsIcon from './assets/icons/settings.png'
import donateIcon from './assets/icons/donate.png'
import lightModeIcon from './assets/icons/light-mode.png'
import guideIcon from './assets/icons/help-guide.png'

/** GuideEntry is one named control: its picture, name and what it does. */
export interface GuideEntry {
  icon: string
  name: string
  text: string
}

/** GuideRule is a rule behind the figures: a claim, then what it means. */
export interface GuideRule {
  title: string
  text: string
}

/** GuideSection is one heading of the Guide. */
export interface GuideSection {
  heading: string
  intro?: string
  entries?: GuideEntry[]
  rules?: GuideRule[]
  paragraphs?: string[]
}

export const guideSections: GuideSection[] = [
  {
    heading: 'The buttons',
    entries: [
      { icon: addIcon, name: 'Add website', text: 'Type an address such as symdiary.com, then Find repositories. Visitron reads the site, lists the GitHub repositories it names and ticks the likely ones; tick the ones whose downloads belong to it.' },
      { icon: editIcon, name: 'Edit website', text: 'Change the selected website\'s address; the site is read again and your ticks are kept where the repository is found again.' },
      { icon: deleteIcon, name: 'Delete website', text: 'Remove the selected website and its download history, after you confirm.' },
      { icon: refreshIcon, name: 'Refresh', text: 'Check every website now rather than waiting for the next scheduled check. While the last check has failed, Refresh carries a warning sign; point at it to read what failed. The sign goes once a check succeeds.' },
      { icon: settingsIcon, name: 'Settings', text: 'Your GoatCounter key, an optional GitHub token, how often to check, starting with Windows and the update check.' },
      { icon: donateIcon, name: 'Donate', text: 'Opens the donation page in your browser.' },
      { icon: lightModeIcon, name: 'Light or dark', text: 'Switches the window between light and dark; the picture is the mode you would move to.' },
      { icon: guideIcon, name: 'Help', text: 'This guide, with About at its foot.' },
    ],
  },
  {
    heading: 'Rules behind the figures',
    rules: [
      { title: 'One download of every macOS disk image is yours.', text: 'You download each release\'s .dmg once to confirm notarisation, so Visitron takes one away from every .dmg file and never goes below nothing.' },
      { title: 'Page loads are visitors per page per day.', text: 'GoatCounter reports how many people read each page each day; one person reading a page twice in a day counts once.' },
      { title: 'A website owns the pages under its own address.', text: 'ernster.dev/WhatDay/ counts its own pages; ernster.dev counts the rest, so nothing is counted twice.' },
      { title: 'Download history starts at the first check.', text: 'GitHub keeps only a running total, so days before Visitron first checked cannot be filled in.' },
      { title: 'A failed check keeps the figures you had.', text: 'Visitron says what failed and when, tries again half an hour later and checks at once when you press Refresh.' },
    ],
  },
  {
    heading: 'Keyboard',
    paragraphs: [
      'Tab or the right arrow moves to the next control; Shift+Tab or the left arrow moves back. Up and Down walk the website list. Enter or Space presses the control you are on; Escape closes a dialog.',
    ],
  },
]
