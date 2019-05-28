// a selection of JS functions for running the GROOT WASM port

// getElementById
function $id(id) {
	return document.getElementById(id);
}

// gotMem sets the webassembly linear memory with the image buffer result at the slice header pointer passed from Go
function gotMem(pointer) {
    memoryBytes.set(bytes, pointer);
}

// prepIndex gets the index ready for loading
function getIndex(indexURL) {
    var reader = new FileReader();
    fetch(indexURL).then(function(response) {
        if (!response.ok) {
            statusUpdate("status", "could not download index!")
        }
        return response.blob();
    }).then(data => {
            reader.readAsArrayBuffer(data);
    })
    .catch(function(error) {
        console.log(error);
    });
    
    reader.onload = (ev) => {
        bytes = new Uint8Array(ev.target.result);
        if (bytes === null) {
            statusUpdate("status", "could not download index!")
        } else {
            initIndexMem(bytes.length);
        }
    }
}

// fastqStreamer reads in a url as a FASTQ data stream, assigns memory for reads and sends them to the groot input channel
/*
function fastqStreamer(fileArr) {
    var files = fileArr[0];
    var funcs = [];
    
    for (var i = 0; i < files.length; i++) {    
        var fn = new Promise(function(resolve) {
            streamFastq(files[i]);
            resolve()
        });
        funcs.push(fn);
    }

    Promise.all(funcs).then(response => {
        // Loop finished, what to do nexT?
        console.log("stream response: ", response)
        })
        .catch(error => {
        // Error
        console.log(error);
        });
}
*/

async function fastqStreamer(fileArr) {
    var items = fileArr[0]
    var closeSignal = false;
    var i = 0;
    await new Promise(async (resolve, reject) => {
        try {
            if (items.length == 0) return resolve();
            let funSync = async () => {
                if ((i+1) == items.length) {
                    closeSignal = true;
                }
                await streamFastq(items[i], closeSignal);
                i++
                if (i == items.length) {
                    resolve();
                }
                else funSync();
            }
            funSync();
        } catch (e) {
            reject(e);
        }
    });
}

function streamFastq(fileName, closeSignal) {
        fetch(fileName).then(function(response) {
            console.log("fetching: ", fileName)
            var pump = function(reader) {
                return reader.read().then(function(result) {
                    // send the chunk on for processing
                    var chunk = result.value;
                    bytes = new Uint8Array(chunk);
                    initFASTQmem(bytes.length);
                    // if we're done reading the stream, return
                    if (result.done) {
                        if (closeSignal == true) {
                            closeFASTQchan();
                        }
                        return;
                    }
                    // go to next chunk via recursion
                    return pump(reader);
                });
            }
            // start reading the response stream
            return pump(response.body.getReader());
        }).catch(function(error) {
            console.log(error);
        });
}

// toggleDiv
function toggleDiv(id) {
    var el = document.getElementById(id);
    if (el.style.display === "block") {
        el.style.display = "none";
    } else {
        el.style.display = "block";
    }
}

//
function statusUpdate(elementID, msg) {
	var element = document.getElementById(elementID);
    $(element).stop().fadeOut(500, function() {
        element.innerHTML = msg;
        $(element, this).fadeIn(500);
    });
}

//
function iconUpdate(id) {
    var el = document.getElementById(id);
    $(el).stop().fadeOut(500, function() {
        el.className = '';
        el.classList.add('fa', 'fa-check');
        el.style.color = '#3fa46a';
        $(el, this).fadeIn(500);
    });
}

//
function setParameters() {
    iconUpdate("paramIcon")
    statusUpdate("status", "parameters are set")
}

function startRecord() {
    var el = document.getElementById("startIcon");
    $(el).stop().fadeOut(500, function() {
        el.className = '';
        el.classList.add('fa', 'fa-compact-disc', "fa-spin");
        el.style.color = '#F16721';
        $(el, this).fadeIn(500);
    });
}
 
function stopRecord() {
    var el = document.getElementById("startIcon");
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

function setButtonColour(id, colour) {
    var el = document.getElementById(id);
    el.style.backgroundColor = colour;
}

function setButtonText(id, txt) {
    var el = document.getElementById(id);
    el.innerHTML = "<span style='color: white'>" + txt + "</span>";
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



// TODO: this is stupid, can't I bypass needing to do so many conversions?
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