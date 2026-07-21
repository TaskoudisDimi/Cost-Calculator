importScripts('https://www.gstatic.com/firebasejs/10.7.1/firebase-app-compat.js')
importScripts('https://www.gstatic.com/firebasejs/10.7.1/firebase-messaging-compat.js')

firebase.initializeApp({
  apiKey: 'AIzaSyDDCBDodg2Lke3JmdrknObAPaVAaBtXskA',
  authDomain: 'billcalculator-fb2dd.firebaseapp.com',
  projectId: 'billcalculator-fb2dd',
  storageBucket: 'billcalculator-fb2dd.firebasestorage.app',
  messagingSenderId: '56920908742',
  appId: '1:56920908742:web:0e4b7ebbf4aca630e548c7',
})

const messaging = firebase.messaging()

messaging.onBackgroundMessage((payload) => {
  const title = payload.notification?.title || 'Cost Calculator'
  const body = payload.notification?.body || ''
  self.registration.showNotification(title, {
    body,
    icon: '/favicon.png',
    badge: '/favicon.png',
    data: payload.data,
    tag: payload.data?.bill_id || 'bill-reminder',
  })
})
