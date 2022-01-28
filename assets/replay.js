// create a xterm for replay
function createReplayTerminal() {
  // vscode-snazzy https://github.com/Tyriar/vscode-snazzy
  // copied from xterm.js website
  var baseTheme = {
    foreground: '#eff0eb',
    background: '#282a36',
    selection: '#97979b33',
    black: '#282a36',
    brightBlack: '#686868',
    red: '#ff5c57',
    brightRed: '#ff5c57',
    green: '#5af78e',
    brightGreen: '#5af78e',
    yellow: '#f3f99d',
    brightYellow: '#f3f99d',
    blue: '#57c7ff',
    brightBlue: '#57c7ff',
    magenta: '#ff6ac1',
    brightMagenta: '#ff6ac1',
    cyan: '#9aedfe',
    brightCyan: '#9aedfe',
    white: '#f1f1f0',
    brightWhite: '#eff0eb'
  };

  const term = new Terminal({
    fontFamily: `'Fira Code', ui-monospace,SFMono-Regular,'SF Mono',Menlo,Consolas,'Liberation Mono',monospace`,
    fontSize: 12,
    theme: baseTheme,
    convertEol: true,
    cursorBlink: true,
  });

  term.open(document.getElementById('terminal_view'));
  term.resize(120, 36);

  const weblinksAddon = new WebLinksAddon.WebLinksAddon();
  term.loadAddon(weblinksAddon);

  // fit the xterm viewpoint to parent element
  const fitAddon = new FitAddon.FitAddon();
  term.loadAddon(fitAddon);
  fitAddon.fit();

  return term;
}

// sleep for ms seconds
function _sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// we could sleep for a long time
// periodically check if we need to end replay. 
// This is pretty ugly but the callback mess otherwise
async function sleep(ms, paused) {
  var loop_cnt = parseInt(ms / 20) + 1

  for (i = 0; i < loop_cnt; i++) {
    if (paused()) {
      return paused()
    }

    await _sleep(20)
  }

  return paused()
}
// convert data to uint8array, we cannot convert it to string as 
// it will mess up special characters
function base64ToUint8array(base64) {
  var raw = window.atob(base64);
  var rawLength = raw.length;
  var array = new Uint8Array(new ArrayBuffer(rawLength));

  for (i = 0; i < rawLength; i++) {
    array[i] = raw.charCodeAt(i);
  }
  return array;
}

// replay session
// term: xterm, path: session file to replay,
// start: start position to replay in percentile, range 0-100
// paused: callback whether to stop play
// prog: callback to update the progress bar
// shift: callback whether we should change position
// end: callback when playback is finished

async function replay_session(term, records, max_wait, total_dur, start, paused, prog, end) {
  var cur = 0

  start = parseInt(total_dur * start / 100)
  term.reset()

  for (const item of records) {
    // we will blast through the beginning of the session
    if (cur >= start) {
      // we are cheating a little bit here, we do not want to wait for too long
      exit = await sleep(Math.min(item.Duration, max_wait), paused)

      if (exit) {
        return
      }
    }

    term.write(base64ToUint8array(item.Data))
    cur += item.Duration

    if (cur > start) {
      prog(parseInt(cur * 100 / total_dur))
    }
  }

  end()
}

function forwardScreen(term, records, end) {
  var cur = 0
  end = parseInt(total_dur * end / 100)
  term.reset()

  for (const item of records) {
    // we will blast through the beginning of the session
    if (cur >= end) {
      return
    }

    term.write(base64ToUint8array(item.Data))
    cur += item.Duration
  }
}


async function fetchAndParse(path, update) {
  var records
  var total_dur = 0

  // read file from server, we need to await twice.
  var res = await fetch(path)

  if (!res.ok) {
    return
  }

  records = await res.json()

  //calculate the total duration
  for (const item of records) {
    item.Duration = parseInt(item.Duration / 1000000)
    total_dur += item.Duration
  }

  console.log("total_dur", total_dur)
  update(records, total_dur)
}


function Init() {
  let term = createReplayTerminal();
  var str = [
    ' ┌────────────────────────────────────────────────────────────────────────────┐\n',
    ' │                 \u001b[32;1mhttps://github.com/syssecfsu/witty\x1b[0m <- click it!            │\n',
    ' └────────────────────────────────────────────────────────────────────────────┘\n',
    ''
  ].join('');

  term.writeln(str);

  // adjust the progress bar size to that of terminal
  var view = document.querySelector("#terminal")
  var pbar = document.querySelector("#replay-control")
  pbar.setAttribute("style", "width:" + (view.offsetWidth - 32) + "px");

  return term
}