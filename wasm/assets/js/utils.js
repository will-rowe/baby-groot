// a selection of JS functions for running the GROOT WASM port

// getElementById
function $id(id) {
	return document.getElementById(id);
}

// gotMem sets the webassembly linear memory with the image buffer result at the slice header pointer passed from Go
function gotMem(pointer) {
    memoryBytes.set(bytes, pointer);
}

// function to read whole file into memory (note: not currently used)
function loadFASTQfiles(f) {
    let reader = new FileReader();
    reader.onload = (ev) => {
      bytes = new Uint8Array(ev.target.result);
      initFASTQmem(bytes.length);
    };
    reader.readAsArrayBuffer(f);
  }

// fastqStreamer reads in a url as a FASTQ data stream, assigns memory for reads and sends them to the groot input channel
function fastqStreamer() {
    //var url = "./groot-files/test-reads-OXA90-100bp-50x-with-errors.fastq";
    var url = "./groot-files/test-data/1.fastq";
    var progress = 0;
    var contentLength = 0;

    var partialCell = '';
    var decoder = new TextDecoder();

    fetch(url).then(function(response) {
        // get the size of the request via the headers of the response
        contentLength = response.headers.get('Content-Length');
        var pump = function(reader) {
            return reader.read().then(function(result) {

                partialCell += decoder.decode(result.value || new Uint8Array, {stream: !result.done});

                
                // report our current progress
                //progress += chunk.byteLength;
                //console.log(((progress / contentLength) * 100) + '%');
                console.log("working on chunk");

                  // Split what we have into CSV 'cells'
                  var cellBoundry = /(\r\n)/;
                  var completeCells = partialCell.split(cellBoundry);
            
                  if (!result.done) {
                    // Last cell is likely incomplete
                    // Keep hold of it for next time
                    partialCell = completeCells[completeCells.length - 1];
                    // Remove it from our complete cells
                    completeCells = completeCells.slice(0, -1);
                  }
            
                  // for each chunk of data, split it by line
                  for (var cell of completeCells) {
                    var lines = cell.split("\n");
                    var arrayLength = lines.length;
                    for (var i = 0; i < arrayLength; i++) {

                        l = toUTF8Array(lines[i])
                        // send the line
                        bytes = new Uint8Array(l);
                        initFASTQmem(bytes.length);
                    }
                  }
            
                // if we're done reading the stream, return
                if (result.done) {
                    closeFASTQchan()
                    return;
                }

                // go to next chunk via recursion
                return pump(reader);
            });
        }

        // start reading the response stream
        return pump(response.body.getReader());
    })
    .catch(function(error) {
        console.log(error);
    });
}








function toggleDiv(id) {
    var el = document.getElementById(id);
    if (el.style.display === "block") {
        el.style.display = "none";
    } else {
        el.style.display = "block";
    }
}

function statusUpdate(elementID, msg) {
	var element = document.getElementById(elementID);
    $(element).stop().fadeOut(500, function() {
        element.innerHTML = msg;
        $(element, this).fadeIn(500);
    });
}

function iconUpdate(id) {
    var el = document.getElementById(id);
    $(el).stop().fadeOut(500, function() {
        el.className = '';
        el.classList.add('fa', 'fa-check');
        el.style.color = '#80ff00';
        $(el, this).fadeIn(500);
    });
}

function setParameters() {
    iconUpdate("paramIcon")
    statusUpdate("status", "parameters are set")
}

function startSpinner() {
    var el = document.getElementById("start");
    $(el).stop().fadeOut(500, function() {
        el.className = '';
        el.classList.add('fa', 'fa-compact-disc', "fa-spin");
        el.style.color = '#F16721';
        $(el, this).fadeIn(500);
    });
}
 
function stopSpinner() {
    var el = document.getElementById("start");
    $(el).stop().fadeOut(500, function() {
        el.className = '';
        el.classList.add('fa', 'fa-play');
        el.style.color = '#E9DBC5';
        $(el, this).fadeIn(500);
    });
}

function startLogo() {
    var svgObj = document.getElementById("logo-animated").contentDocument;
    var l1 = svgObj.getElementById("leaf-1");
    var l2 = svgObj.getElementById("leaf-2");
    var l3 = svgObj.getElementById("leaf-3");    
    l1.classList.add('growing');
    l2.classList.add('growing');
    l3.classList.add('growing');
}

function stopLogo() {
    var svgObj = document.getElementById("logo-animated").contentDocument;
    var l1 = svgObj.getElementById("leaf-1");
    var l2 = svgObj.getElementById("leaf-2");
    var l3 = svgObj.getElementById("leaf-3");    
    l1.classList.remove('growing');
    l2.classList.remove('growing');
    l3.classList.remove('growing');
}

function addResults(ref, abun) {
    $("#resultsContent").append("<tr><td>" + ref + "</td><td>" + abun + "</td></tr>");
}

function updateTimer(elapsedTime) {
    $("#runTime").append("<sub style='color:black;'>time elapsed: " + elapsedTime + "</sub>");
}

function getIndex(url) {
    var reader = new FileReader();
    fetch(url)
    .then(function(response) {
        if (!response.ok) {
            setButtonColour("indexGetter", "red")
            setButtonText("indexGetter", "download failed");
        }
        return response.blob();
    })
    .then(data => {
        reader.readAsArrayBuffer(data);
    });
    reader.onload = (ev) => {
        bytes = new Uint8Array(ev.target.result);
        if (bytes === null) {
            setButtonColour("indexGetter", "red")
            setButtonText("indexGetter", "download failed");
        } else {
            setButtonColour("indexGetter", "#80ff00");
            setButtonText("indexGetter", "downloaded!");
        }
        initIndexMem(bytes.length);
    };
}

function setButtonColour(id, colour) {
    var el = document.getElementById(id);
    el.style.backgroundColor = colour;
}

function setButtonText(id, txt) {
    var el = document.getElementById(id);
    el.innerHTML = "<span>" + txt + "</span>";
}

$('#uploader1').bind('change', function () {
    var filename = $("#uploader1").val();
    if (/^\s*$/.test(filename)) {
      $("#fastqSelecter").removeClass('active');
      $("#noFile").text("No file chosen..."); 
    }
    else {
      $("#fastqSelecter").addClass('active');
      $("#noFile").text(filename.replace("C:\\fakepath\\", "")); 
    }
  });



function toUTF8Array(str) {
    var utf8 = [];
    for (var i=0; i < str.length; i++) {
        var charcode = str.charCodeAt(i);
        if (charcode < 0x80) utf8.push(charcode);
        else if (charcode < 0x800) {
            utf8.push(0xc0 | (charcode >> 6), 
                      0x80 | (charcode & 0x3f));
        }
        else if (charcode < 0xd800 || charcode >= 0xe000) {
            utf8.push(0xe0 | (charcode >> 12), 
                      0x80 | ((charcode>>6) & 0x3f), 
                      0x80 | (charcode & 0x3f));
        }
        // surrogate pair
        else {
            i++;
            // UTF-16 encodes 0x10000-0x10FFFF by
            // subtracting 0x10000 and splitting the
            // 20 bits of 0x0-0xFFFFF into two halves
            charcode = 0x10000 + (((charcode & 0x3ff)<<10)
                      | (str.charCodeAt(i) & 0x3ff));
            utf8.push(0xf0 | (charcode >>18), 
                      0x80 | ((charcode>>12) & 0x3f), 
                      0x80 | ((charcode>>6) & 0x3f), 
                      0x80 | (charcode & 0x3f));
        }
    }
    return utf8;
}