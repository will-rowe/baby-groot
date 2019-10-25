// getGraphs gets the groot graphs ready for loading
function getGraphs(graphURL) {
    var reader = new FileReader();
    fetch(graphURL).then(function(response) {
        if (!response.ok) {
            statusUpdate("status", "could not download groot graphs!")
        }
        return response.blob();
    }).then(data => {
            reader.readAsArrayBuffer(data);
    })
    .catch(function(error) {
        console.log(error);
    });
    reader.onload = (ev) => {
        var raw_data = new Uint8Array(ev.target.result, 0, ev.target.result.byteLength);
        if (raw_data === null) {
            statusUpdate("status", "could not download groot graphs!")
        } else {
            loadGraphs(graphURL, raw_data, reader.result.byteLength);
        }
    }
}

// getLSHforest gets the index ready for loading
function getLSHforest(lshfURL) {
    var reader = new FileReader();
    fetch(lshfURL).then(function(response) {
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
        var raw_data = new Uint8Array(ev.target.result, 0, ev.target.result.byteLength);
        if (raw_data === null) {
            statusUpdate("status", "could not download index!")
        } else {
            loadIndex(lshfURL, raw_data, reader.result.byteLength);
        }
    }
}

// fastqStreamer
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

// streamFastq is called by the fastqStreamer
function streamFastq(fileName, closeSignal) {
        fetch(fileName).then(function(response) {
            console.log("fetching: ", fileName)
            var pump = function(reader) {
                return reader.read().then(function(result) {
                    // send the chunk on for processing
                    var raw_data = new Uint8Array(result.value);
                    if (raw_data === null) {
                        statusUpdate("status", "could not munch FASTQ!");
                    } else {
                        loadFASTQ(fileName, raw_data, raw_data.length);
                    }
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

// getElementById
function $id(id) {
	return document.getElementById(id);
}

// toggleDiv
function toggleDiv(id) {
    var el = document.getElementById(id);
    if (el.style.display == 'block') {
        el.style.display = 'none';
    } else {
        el.style.display = 'block';
    }
    console.log(el);
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
