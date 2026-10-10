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

/** GuideStep is one numbered step, with any lines to copy exactly. */
export interface GuideStep {
  text: string
  code?: string[]
}

/** GuideSection is one heading of the Guide. */
export interface GuideSection {
  heading: string
  intro?: string
  steps?: GuideStep[]
  entries?: GuideEntry[]
  rules?: GuideRule[]
  paragraphs?: string[]
}

// The GoatCounter tag every counted page carries (Amendment 14). The first
// line makes GoatCounter record each page with its website, which is how
// Visitron tells one website from another; it must come before the second.
// youraccount stands for the reader's own account name: Visitron names no
// one's account.
const goatCounterTag = [
  '<script>window.goatcounter = {path: function (p) { return location.host + p }}</script>',
  '<script data-goatcounter="https://youraccount.goatcounter.com/count" async src="https://gc.zgo.at/count.js"></script>',
]

export const guideSections: GuideSection[] = [
  {
    heading: 'Before you start: GoatCounter',
    intro: 'Page loads come from GoatCounter, a web statistics service, so they need your own GoatCounter account and a tag on every page you want counted. Downloads need neither: they come from GitHub.',
    steps: [
      { text: 'Sign up at goatcounter.com. The account name you choose there becomes your address: youraccount.goatcounter.com.' },
      {
        text: 'Put these two lines in the head of every page you want counted, in this order, with youraccount replaced by your account name:',
        code: goatCounterTag,
      },
      { text: 'Publish the pages, open one, then open your GoatCounter dashboard: the visit shows within seconds. An ad blocker in your own browser can stop it being counted.' },
      { text: 'In Visitron, open Settings and enter your GoatCounter account name, then an API key; the steps for each are under its field.' },
    ],
  },
  {
    heading: 'The buttons',
    entries: [
      { icon: addIcon, name: 'Add website', text: 'Type an address such as example.com, then Find repositories. Visitron reads the site, lists the GitHub repositories it names and ticks the likely ones; tick the ones whose downloads belong to it.' },
      { icon: editIcon, name: 'Edit website', text: 'Change the selected website\'s address; the site is read again and your ticks are kept where the repository is found again.' },
      { icon: deleteIcon, name: 'Delete website', text: 'Remove the selected website and its download history, after you confirm.' },
      { icon: refreshIcon, name: 'Refresh', text: 'Check every website now rather than waiting for the next scheduled check. While the last check has failed, Refresh carries a warning sign; point at it to read what failed. The sign goes once a check succeeds.' },
      { icon: settingsIcon, name: 'Settings', text: 'Your GoatCounter account name and key, an optional GitHub token, how often to check, starting with Windows and the update check. Each field says where its key or token comes from.' },
      { icon: donateIcon, name: 'Donate', text: 'Opens the donation page in your browser.' },
      { icon: lightModeIcon, name: 'Light or dark', text: 'Switches the window between light and dark; the picture is the mode you would move to.' },
      { icon: guideIcon, name: 'Help', text: 'A menu holding this guide, About, the licence and Check for updates, which asks GitHub for a newer Visitron there and then. With the update check on in Settings, Visitron also asks by itself shortly after it opens and once a day. Down, Enter or Space opens the menu; Up and Down move through it; Escape closes it.' },
    ],
  },
  {
    heading: 'Rules behind the figures',
    rules: [
      { title: 'One download of every macOS disk image is yours.', text: 'You download each release\'s .dmg once to confirm notarisation, so Visitron takes one away from every .dmg file and never goes below nothing.' },
      { title: 'Page loads are visitors per page per day.', text: 'GoatCounter reports how many people read each page each day; one person reading a page twice in a day counts once.' },
      { title: 'A website owns the pages under its own address.', text: 'example.com/app/ counts its own pages; example.com counts the rest, so nothing is counted twice.' },
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
