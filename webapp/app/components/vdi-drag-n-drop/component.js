import Ember from 'ember';

export default Ember.Component.extend({

  /* global $:false */

  classNames: ['vdi-drag-n-drop'],
  classNameBindings: ['show:state-show:state-hide'],
  session: Ember.inject.service('session'),
  flow: null,
  progress: null,
  show: false,
  queue: [],
  state: null,

  reset: function() {
    $('body').off('dragenter dragover');
    this.set('queue', []);
  }.on('init'),

  showElement() {
    this.set('show', true);
  },

  hideElement() {
    this.set('show', false);
  },

  dragLeave() {
    this.set('dragAndDropActive', false);
    this.hideElement();
  },

  drop() {
    this.set('dragAndDropActive', false);
    this.hideElement();
  },

  fileExistInFlowQueue(file) {
    var flowfiles = this.get('flow.files');
    for (var i = 0; i < flowfiles.length; i++) {
      if (flowfiles[i].name === file.name) {
        return true;
      }
    }
    return false;
  },

  flowfileExistInQueue(file) {
    var queue = this.get('queue');
    for (var i = 0; i < queue.length; i++) {
      if (queue[i].content.name === file.name) {
        return true;
      }
    }
    return false;
  },

  updateQueue() {

    var queue = this.get('queue');
    var flowQueue = this.get('flow.files');

    for ( var i = 0; i < queue.length; i++) {
      if (!this.fileExistInFlowQueue(queue[i].content)) {
        this.get('queue').removeObject(queue[i]);
      }
    }

    for ( var j = 0; j < flowQueue.length; j++) {
      if (this.flowfileExistInQueue(flowQueue[j])) {
        Ember.set(this.get('queue').objectAt(j).content, "current_progress", flowQueue[j].progress());
      }
      else {
        var obj = Ember.ObjectProxy.create({ content : flowQueue[j]});
        this.get('queue').pushObject(obj);
      }
    }
  },

  removeCompleteDownload() {

    var flowfiles = this.get('flow.files');
    if (flowfiles === null) {
      return ;
    }
    var i = flowfiles.length;
    while (--i >= 0) {
      if (flowfiles[i].current_progress === 1) {
        this.get('flow.files').removeAt(i);
        this.get('queue').removeAt(i);
      }
    }
  },

  didInsertElement() {

    $('body').on('dragenter dragover', function() {
      this.set('show', true);
      if (this.get('dragAndDropActive') === false) {
        this.set('dragAndDropActive', true);
        this.showElement();
      }
    }.bind(this));

    this.set('flow', new window.Flow({
      target: '/upload',
      headers: { Authorization : "Bearer " + this.get('session.access_token') },
      singleFile: false,
      allowDuplicateUploads: false,
      supportDirectory: false,
    }));

    this.get('flow').assignDrop(this.element);
    this.get('flow').assignBrowse($('.browse'));

    this.get('flow').on('filesSubmitted', () => {
      this.$().find('.state').show();
      this.updateQueue();
      this.get('flow').upload();
    });

    this.get('flow').on('complete', () => {

      this.updateQueue();
      if (this.get('flow').progress() === 1) {
        this.downloadCompleted();
        this.sendAction('complete');
      }
    });

    this.get('flow').on('fileSuccess', () => {

      this.updateQueue();
      this.sendAction('complete');
    });

    this.get('flow').on('error', () => {
        this.set('state', "Error");
    });

    this.get('flow').on('progress', () => {

      this.updateQueue();
      this.set('progress', this.get('flow').progress());
    });
  },

  downloadCompleted() {
    this.set('progress', null);
    this.updateQueue();

    if (this.get('state') !== 'Aborted') {
      this.toast.success("Upload successful");
      this.set('state', "Completed");
      setTimeout(() => {
        $('.state').fadeOut(700, () => {
          this.set('state', null);
          $('.state').fadeIn(0);
        });
      }, 3000);
    }
    else {
      this.set('state', null);
    }
  },

  stopUpload() {
    this.toast.info("Abort successful");
    this.set('state', "Aborted");
    this.get('flow').cancel();
    this.downloadCompleted();
  },

  actions: {
    cancelUpload() {
      this.stopUpload();
    },

    flushHistory() {
      this.removeCompleteDownload();
    },

    cancelSingleUpload() {
      this.updateQueue();
      if (this.get('queue').length === 0) {
        this.stopUpload();
      }
    }
  },
});
