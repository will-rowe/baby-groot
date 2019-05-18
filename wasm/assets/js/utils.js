// gotMem sets the webassembly linear memory with the image buffer result at the slice header pointer passed from Go
function gotMem1(pointer) {
	memoryBytes.set(bytes, pointer);
	// load the file
	loadFastq();
}

// gotMem sets the webassembly linear memory with the image buffer result at the slice header pointer passed from Go
function gotMem2(pointer) {
	memoryBytes.set(bytes, pointer);
	// load the file
	loadIndex();
}

// displayImage takes the pointer to the target image in the wasm linear memory and its length. Gets the resulting byte slice and creates an image blob.
function displayImage(pointer, length) {
	let resultBytes = memoryBytes.slice(pointer, pointer + length);
	let blob = new Blob([resultBytes], {'type': imageType});
	document.getElementById('targetImg').src = URL.createObjectURL(blob);
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

function toggleDiv(divID) {
    var el = document.getElementById(divID);
    if (el.style.display === "none") {
      el.style.display = "block";
    } else {
      el.style.display = "none";
    }
}

function toggleModal() {
    document.querySelector(".modal").classList.toggle("show-modal");
}

function setParameters() {
    iconUpdate("paramIcon")
    toggleModal()
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
        el.style.color = '#FFF';
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
