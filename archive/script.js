document.addEventListener('DOMContentLoaded', (event) => {
const url = "signup.link"
fetch(url)
.then( r => r.text() )
.then( data =>
     document.getElementById('signup').innerHTML = `<a href="${data}">Sign Up</a>`
)
});

function setDarkMode(d) {
  var html = document.documentElement;
  html.className = d ? 'dark' : '';
  console.log(`HTML is now ${html.className} for ${d}`);
}

setDarkMode(window.matchMedia &&
            window.matchMedia('(prefers-color-scheme: dark)').matches);

window.matchMedia('(prefers-color-scheme: dark)')
    .addEventListener('change', event => {
      if (event.matches) {
        setDarkMode(true);
      }
    });

window.matchMedia('(prefers-color-scheme: light)')
    .addEventListener('change', event => {
      if (event.matches) {
        setDarkMode(false);
      }
    });
