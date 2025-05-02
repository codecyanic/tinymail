/*
 * TinyMail - Minimalistic email client
 * Copyright (C) 2025 Cyanic
 *
 * This program is free software: you can redistribute it and/or modify it
 * under the terms of the GNU Affero General Public License as published by the
 * Free Software Foundation, either version 3 of the License, or (at your
 * option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
 * FITNESS FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License
 * for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

const pageSize = 25

const compose = (fetchOptions, emailFrom, emailTo='', emailSubject='') => {
    const compose = document.createElement('div')
    compose.className = 'compose'

    const overlay = document.createElement('div')
    overlay.className = 'overlay'
    overlay.append(compose)

    overlay.onclick = e => {
        if (e.target == overlay)
            overlay.remove()
    }

    const send = document.createElement('div')
    send.textContent = 'Send'
    send.className = 'send-btn btn'

    const close = document.createElement('div')
    close.textContent = 'Close'
    close.className = 'close-btn btn'
    close.onclick = () => overlay.remove()

    const titleBar = document.createElement('div')
    titleBar.className = 'title-bar'
    titleBar.append(send, close)

    const fromLabel = document.createElement('span')
    fromLabel.className = 'message-label'
    fromLabel.textContent = 'From:'
    const fromInput = document.createElement('input')
    fromInput.value = emailFrom

    const toLabel = document.createElement('span')
    toLabel.className = 'message-label'
    toLabel.textContent = 'To:'
    const toInput = document.createElement('input')
    toInput.value = emailTo

    const subjectLabel = document.createElement('span')
    subjectLabel.className = 'message-label'
    subjectLabel.textContent = 'Subject:'
    const subjectInput = document.createElement('input')
    subjectInput.value = emailSubject

    const composeHeader = document.createElement('div')
    composeHeader.className = 'compose-header'
    composeHeader.append(
        fromLabel, fromInput,
        toLabel, toInput,
        subjectLabel, subjectInput,
    )

    const composeBody = document.createElement('textarea')
    composeBody.className = 'compose-body'

    compose.append(titleBar, composeHeader, composeBody)

    send.onclick = () => {
        if (send.locked)
            return

        const data = {
            from: fromInput.value,
            to: toInput.value,
            subject: subjectInput.value,
            body: composeBody.value,
        }

        const options = {
            ...fetchOptions,
            method: 'POST',
            body: JSON.stringify(data)
        }
        fetch(`/api/send`, options)
            .then(r => {
                if (r.ok)
                    overlay.remove()
            })
            .finally(() => {
                send.locked = false
            })
    }

    return overlay
}

const initApp = (email, data, fetchOptions) => {
    const app = document.createElement('div')
    const mailboxes = document.createElement('div')
    mailboxes.className = 'mailboxes'
    const messages = document.createElement('div')
    messages.className = 'messages'
    const message = document.createElement('div')
    message.className = 'message'
    const newMessage = document.createElement('div')
    newMessage.className = 'new-message-btn btn'
    newMessage.textContent = 'New Message'
    newMessage.onclick = () => app.append(compose(fetchOptions, email))

    const messageButton = (mailbox, msg) => {
        const btn = document.createElement('div')
        btn.className = 'message-item selectable'
        btn.textContent = msg.subject
        btn.title = msg.subject
        if (!msg.seen)
            btn.classList.add('unread')
        btn.onclick = () => {
            if (messages.locked)
                return
            messages.locked = true

            messages.childNodes.forEach(
                btn => btn.classList.remove('selected')
            )
            btn.classList.add('selected')
            message.innerHTML = ''
            fetch(`/api/mailbox/${mailbox.name}/message/${msg.uid}`, fetchOptions)
                .then(r => r.json())
                .then(data => {
                    const from = document.createElement('div')
                    const fromLabel = document.createElement('span')
                    fromLabel.className = 'message-label'
                    fromLabel.textContent = 'From: '
                    const fromText = document.createElement('span')
                    fromText.textContent = data.from
                    from.append(fromLabel, fromText)

                    const subject = document.createElement('div')
                    const subjectLabel = document.createElement('span')
                    subjectLabel.className = 'message-label'
                    subjectLabel.textContent = 'Subject: '
                    const subjectText = document.createElement('span')
                    subjectText.textContent = data.subject
                    subject.append(subjectLabel, subjectText)

                    const body = document.createElement('div')
                    body.textContent = data.body
                    const pre = document.createElement('pre')
                    pre.className = 'message-body'
                    pre.append(body)

                    message.append(from, subject, pre)

                    // TODO: Remove after IDLE is implemented
                    btn.classList.remove('unread')
                })
                .finally(() => {
                    messages.locked = false
                })
        }
        return btn
    }

    const mailboxButton = mailbox => {
        const btn = document.createElement('div')
        btn.className = 'mailbox-item selectable'
        btn.textContent = mailbox.name === 'INBOX' ? 'Inbox' : mailbox.name
        const onclick = async () => {
            mailboxes.childNodes.forEach(
                btn => btn.classList.remove('selected')
            )
            btn.classList.add('selected')
            message.innerHTML = ''
            messages.innerHTML = ''
            if (mailbox.messages == null) {
                await fetch(`/api/mailbox/${mailbox.name}`, fetchOptions)
                    .then(r => r.json())
                    .then(data => {
                        mailbox.uids = data.uids
                        mailbox.messages = data.messages
                    })
                    .catch(e => {
                        mailbox.locked = false
                        return Promise.reject(e)
                    })
            }

            const removeMessageUIDs = messages => {
                const processed = new Set()
                messages.forEach(msg => {
                    processed.add(msg.uid)
                })
                mailbox.uids = mailbox.uids.filter(uid => !processed.has(uid))
            }

            const more = document.createElement('div')
            more.className = 'load-more-btn btn'
            more.textContent = 'Load more'
            more.onclick = () => {
                if (mailbox.locked)
                    return
                mailbox.locked = true

                const uids = mailbox.uids.slice(0, pageSize).join()
                fetch(`/api/mailbox/${mailbox.name}/messages/${uids}`, fetchOptions)
                    .then(r => r.json())
                    .then(data => {
                        more.remove()
                        data.forEach(msg => {
                            mailbox.messages.push(msg)
                            messages.append(messageButton(mailbox, msg))
                        })
                        removeMessageUIDs(data)
                        if (mailbox.uids.length)
                            messages.append(more)
                    })
                    .finally(() => {
                        mailbox.locked = false
                    })
            }

            mailbox.messages.forEach(msg => {
                messages.append(messageButton(mailbox, msg))
            })
            removeMessageUIDs(mailbox.messages)
            if (mailbox.uids.length)
                messages.append(more)

            mailbox.locked = false
        }

        btn.onclick = () => {
            if (mailbox.locked)
                return
            mailbox.locked = true
            onclick().finally(
                () => mailbox.locked = false
            )
        }
        return btn
    }

    data.mailboxes.forEach(mailbox => {
        mailboxes.append(mailboxButton(mailbox))
    })

    app.className = 'app'
    app.append(newMessage, messages, message, mailboxes)
    document.body.innerHTML = ''
    document.body.append(app)

    mailboxes.childNodes[0].click()
}

const initLogin = () => {
    const h = document.createElement('h1')
    h.textContent = 'TinyMail login'

    const email = document.createElement('input')
    email.type = 'email'
    email.placeholder = 'email'
    const password = document.createElement('input')
    password.type = 'password'
    password.placeholder = 'password'

    const submit = document.createElement('input')
    submit.type = 'submit'
    submit.value = 'Login'

    const form = document.createElement('form')
    form.className = 'login-form'
    form.append(h, email, password, submit)


    form.onsubmit = () => {
        const credentials = btoa(`${email.value}:${password.value}`)
        const opt = {
            headers: {
                'Authorization': `Basic ${credentials}`,
            },
        }

        fetch('/api/account', opt)
            .then(r => r.json())
            .then(data => initApp(email.value, data, opt))
        return false
    }

    const footer = document.createElement('footer')
    footer.innerHTML = `© 2025 Cyanic –
    <a href="/agpl-3.0-standalone.html">License</a> –
    <a href="/source.tgz">Source</a>`

    document.body.innerHTML = ''
    document.body.append(form, footer)
}

initLogin()
