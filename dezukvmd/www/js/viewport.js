/*
    viewport.js
*/
let massStorageSwitchURL = "/api/v1/mass_storage/switch"; //side accept kvm or remote

// CSRF-protected AJAX function
$.cjax = function(payload){
    let requireTokenMethod = ["POST", "PUT", "DELETE"];
    if (requireTokenMethod.includes(payload.method) || requireTokenMethod.includes(payload.type)){
        //csrf token is required
        let csrfToken = document.getElementsByTagName("meta")["dezukvm.csrf.token"].getAttribute("content");
        payload.headers = {
            "dezukvm_csrf_token": csrfToken,
        }
    }

    $.ajax(payload);
}

$(document).ready(function() {
    // Check if the user has opted out of seeing the audio tips
    if (localStorage.getItem('dontShowAudioTipsAgain') === 'true') {
        $('#audioTips').remove();
    }
});

/* Mass Storage Switch */
function switchMassStorageToRemote(){
    $.cjax({
        url: massStorageSwitchURL,
        type: 'POST',
        data: {
            side: 'remote',
            uuid: kvmDeviceUUID
        },
        success: function(response) {
            if (response.error) {
                alert('Error switching Mass Storage to Remote: ' + response.error);
            }
        },
        error: function(xhr, status, error) {
            alert('Error switching Mass Storage to Remote: ' + error);
        }
    });
}

function switchMassStorageToKvm(){
    $.cjax({
        url: massStorageSwitchURL,
        type: 'POST',
        data: {
            side: 'kvm',
            uuid: kvmDeviceUUID
        },
        success: function(response) {
            if (response.error) {
                alert('Error switching Mass Storage to KVM: ' + response.error);
            }
        },
        error: function(xhr, status, error) {
            alert('Error switching Mass Storage to KVM: ' + error);
        }
    });
}


/*
    UI elements and events
*/

function handleDontShowAudioTipsAgain(){
    localStorage.setItem('dontShowAudioTipsAgain', 'true');
    $('#audioTips').remove();
}

function toggleFullScreen(){
    let elem = document.documentElement;
    if (!document.fullscreenElement) {
        if (elem.requestFullscreen) {
            elem.requestFullscreen();
        } else if (elem.mozRequestFullScreen) { // Firefox
            elem.mozRequestFullScreen();
        } else if (elem.webkitRequestFullscreen) { // Chrome, Safari, Opera
            elem.webkitRequestFullscreen();
        } else if (elem.msRequestFullscreen) { // IE/Edge
            elem.msRequestFullscreen();
        }
    } else {
        if (document.exitFullscreen) {
            document.exitFullscreen();
        } else if (document.mozCancelFullScreen) {
            document.mozCancelFullScreen();
        } else if (document.webkitExitFullscreen) {
            document.webkitExitFullscreen();
        } else if (document.msExitFullscreen) {
            document.msExitFullscreen();
        }
    }
}
