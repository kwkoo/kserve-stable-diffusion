var prompt = null;
var queryButton = null;
var spinner = null;
var modelImage = null;

function initialize() {
    prompt = document.getElementById('prompt');
    queryButton = document.getElementById('query-button');
    spinner = document.getElementById('spinner');
    modelImage = document.getElementById('model-image');
}

function showQueryButton(show) {
    queryButton.style.visibility = (show?'visible':'hidden');
}

function showSpinner(show) {
    spinner.style.display = (show?'block':'none');
}

function showImage(show) {
    modelImage.style.display = (show?'block':'none');
}

function lookForEnter() {
    if (event.key === 'Enter') query();
}

function processLine(text) {
    let obj = null;
    try {
        obj = JSON.parse(text);
    }
    catch (e) {
        return
    }
    if (obj == null) return;
    if (obj.error != null) {
        alert(obj.error);
        return
    }
    if (obj.image != null) {
        modelImage.setAttribute('src', 'data:image/jpeg;charset=utf-8;base64,' + obj.image);
        showImage(true);
    }
}

// Function to read a stream line by line
async function readStreamLineByLine(stream) {
    const reader = stream.getReader();
    const decoder = new TextDecoder();
    let partialLine = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      const chunk = decoder.decode(value, { stream: true });
      const lines = (partialLine + chunk).split('\n');
      partialLine = lines.pop(); // Store incomplete line for the next iteration

      for (const line of lines) {
        // Process each line here
        processLine(line);
      }
    }

    // Process the remaining partial line, if any
    if (partialLine) {
      // Process the last line
      processLine(partialLine);
    }

    reader.releaseLock();
}

function query() {
    showQueryButton(false);
    showSpinner(true);
    showImage(false);
    fetch('/api/infer', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json'},
        body: JSON.stringify({ prompt: prompt.value })
    })
    .then(response => response.body)
    .then(readStreamLineByLine)
    .catch(error => alert(error))
    .finally( () => {
        showSpinner(false);
        showQueryButton(true);
    });
}
