package widget

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/internal/cache"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// errFormItemInitialState defines the error if the initial validation for a FormItem result
// in an error
var errFormItemInitialState = errors.New("widget.FormItem initial state error")

// FormItem provides the details for a row in a form
type FormItem struct {
	Text   string
	Widget fyne.CanvasObject

	// Since: 2.0
	HintText string

	validationError error
	invalid         bool
	helperOutput    *canvas.Text
}

// NewFormItem creates a new form item with the specified label text and input widget
func NewFormItem(text string, widget fyne.CanvasObject) *FormItem {
	return &FormItem{Text: text, Widget: widget}
}

// Form widget is two column grid where each row has a label and a widget (usually an input).
// The last row of the grid will contain the appropriate form control buttons if any should be shown.
// Setting OnSubmit will set the submit button to be visible and call back the function when tapped.
// Setting OnCancel will do the same for a cancel button.
// If you change OnSubmit/OnCancel after the form is created and rendered, you need to call
// Refresh() to update the form with the correct buttons.
// Setting OnSubmit/OnCancel to nil will remove the buttons.
type Form struct {
	BaseWidget

	Items      []*FormItem
	OnSubmit   func()
	OnCancel   func()
	SubmitText string
	CancelText string

	itemGrid     *fyne.Container
	buttonBox    *fyne.Container
	cancelButton *Button
	submitButton *Button
}

// Append adds a new row to the form, using the text as a label next to the specified Widget
func (f *Form) Append(text string, widget fyne.CanvasObject) {
	item := &FormItem{Text: text, Widget: widget}
	f.AppendItem(item)
}

// AppendItem adds the specified row to the end of the Form
func (f *Form) AppendItem(item *FormItem) {
	f.ExtendBaseWidget(f) // could be called before render

	f.Items = append(f.Items, item)
	if f.itemGrid != nil {
		f.itemGrid.Add(f.createLabel(item.Text))
		f.itemGrid.Add(f.createInput(item))
		f.setUpValidation(item.Widget, len(f.Items)-1)
	}

	f.Refresh()
}

// MinSize returns the size that this widget should not shrink below
func (f *Form) MinSize() fyne.Size {
	f.ExtendBaseWidget(f)
	return f.BaseWidget.MinSize()
}

// Refresh updates the widget state when requested.
func (f *Form) Refresh() {
	cache.Renderer(f.super()) // we are about to make changes to renderer created content... not great!
	f.updateButtons()
	f.updateLabels()
	f.BaseWidget.Refresh()
	canvas.Refresh(f.super()) // refresh ourselves for BG color - the above updates the content
}

func (f *Form) createInput(item *FormItem) fyne.CanvasObject {
	_, ok := item.Widget.(fyne.Validatable)
	if item.HintText == "" {
		if !ok {
			return item.Widget
		}
		if e, ok := item.Widget.(*Entry); ok && e.Validator == nil { // we don't have validation
			return item.Widget
		}
	}

	text := canvas.NewText(item.HintText, theme.PlaceHolderColor())
	text.TextSize = theme.CaptionTextSize()
	text.Move(fyne.NewPos(theme.Padding()*2, theme.Padding()*-0.5))
	item.helperOutput = text
	f.updateHelperText(item)
	return fyne.NewContainerWithLayout(layout.NewVBoxLayout(), item.Widget, fyne.NewContainerWithoutLayout(text))
}

func (f *Form) createLabel(text string) *canvas.Text {
	return &canvas.Text{Text: text,
		Alignment: fyne.TextAlignTrailing,
		Color:     theme.ForegroundColor(),
		TextSize:  theme.TextSize(),
		TextStyle: fyne.TextStyle{Bold: true}}
}

func (f *Form) updateButtons() {
	if f.CancelText == "" {
		f.CancelText = "Cancel"
	}
	if f.SubmitText == "" {
		f.SubmitText = "Submit"
	}

	// set visibility on the buttons
	if f.OnCancel == nil {
		f.cancelButton.Hide()
	} else {
		f.cancelButton.SetText(f.CancelText)
		f.cancelButton.OnTapped = f.OnCancel
		f.cancelButton.Show()
	}
	if f.OnSubmit == nil {
		f.submitButton.Hide()
	} else {
		f.submitButton.SetText(f.SubmitText)
		f.submitButton.OnTapped = f.OnSubmit
		f.submitButton.Show()
	}
	if f.OnCancel == nil && f.OnSubmit == nil {
		f.buttonBox.Hide()
	} else {
		f.buttonBox.Show()
	}
}

func (f *Form) checkValidation(err error) {
	if err != nil {
		f.submitButton.Disable()
		return
	}

	for _, item := range f.Items {
		if item.invalid {
			f.submitButton.Disable()
			return
		}
	}

	f.submitButton.Enable()
}

func (f *Form) setUpValidation(widget fyne.CanvasObject, i int) {
	updateValidation := func(err error) {
		if err == errFormItemInitialState {
			return
		}
		f.Items[i].validationError = err
		f.Items[i].invalid = err != nil
		f.checkValidation(err)
		f.updateHelperText(f.Items[i])
	}
	if w, ok := widget.(fyne.Validatable); ok {
		f.Items[i].invalid = w.Validate() != nil
		if e, ok := w.(*Entry); ok {
			e.onFocusChanged = func(bool) {
				updateValidation(e.validationError)
			}
			if e.Validator != nil && f.Items[i].invalid {
				// set initial state error to guarantee next error (if triggers) is always different
				e.SetValidationError(errFormItemInitialState)
			}
		}
		w.SetOnValidationChanged(updateValidation)
	}
}

func (f *Form) updateHelperText(item *FormItem) {
	if item.helperOutput == nil {
		return // testing probably, either way not rendered yet
	}
	showHintIfError := false
	if e, ok := item.Widget.(*Entry); ok && (!e.dirty || e.focused) {
		showHintIfError = true
	}
	if item.validationError == nil || showHintIfError {
		item.helperOutput.Text = item.HintText
		item.helperOutput.Color = theme.PlaceHolderColor()
	} else {
		item.helperOutput.Text = item.validationError.Error()
		item.helperOutput.Color = theme.ErrorColor()
	}
	item.helperOutput.Refresh()
}

func (f *Form) updateLabels() {
	for i, item := range f.Items {
		l := f.itemGrid.Objects[i*2].(*canvas.Text)
		l.TextSize = theme.TextSize()
		if dis, ok := item.Widget.(fyne.Disableable); ok {
			if dis.Disabled() {
				l.Color = theme.DisabledColor()
			} else {
				l.Color = theme.ForegroundColor()
			}
		}

		l.Text = item.Text
		l.Refresh()
		f.updateHelperText(item)
	}
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (f *Form) CreateRenderer() fyne.WidgetRenderer {
	f.ExtendBaseWidget(f)

	f.cancelButton = &Button{Icon: theme.CancelIcon(), OnTapped: f.OnCancel}
	f.submitButton = &Button{Icon: theme.ConfirmIcon(), OnTapped: f.OnSubmit, Importance: HighImportance}
	f.buttonBox = fyne.NewContainerWithLayout(layout.NewHBoxLayout(), layout.NewSpacer(), f.cancelButton, f.submitButton)

	objects := make([]fyne.CanvasObject, len(f.Items)*2)
	for i, item := range f.Items {
		objects[i*2] = f.createLabel(item.Text)

		f.setUpValidation(item.Widget, i)
		objects[i*2+1] = f.createInput(item)
	}
	f.itemGrid = fyne.NewContainerWithLayout(layout.NewFormLayout(), objects...)

	renderer := &simpleRenderer{content: fyne.NewContainerWithLayout(layout.NewVBoxLayout(), f.itemGrid, f.buttonBox)}
	f.updateButtons()
	f.updateLabels()
	f.checkValidation(nil) // will trigger a validation check for correct intial validation status
	return renderer
}

// NewForm creates a new form widget with the specified rows of form items
// and (if any of them should be shown) a form controls row at the bottom
func NewForm(items ...*FormItem) *Form {
	form := &Form{Items: items}
	form.ExtendBaseWidget(form)

	return form
}
