// Package errsel builds on the concept of error causers in
// github.com/pkg/errors by extending errors with a particular
// set of classes.
//
// This package also provides an efficient, flexible method for
// querying error conditions in the form of selectors. Error
// classes from this package are themselves selectors, and provide
// an abstract, structured approach to errors that is fully
// compatible with common go error handling idioms.
//
// In essence, classes are a mechanism of abstraction over error
// types, hopefully one more pleasant to work with than a complex
// series of boolean expressions and branch statements.
//
// For example, the following (which may be duplicated in many places):
//
//    func something() {
//        err := someJankyFunction()
//        if err != nil && errors.Cause(err) != sql.ErrNoRows {
//            if errors.Cause(err) == sql.ErrConflict || errors.Cause(err) == sql.ErrTransactionClosed {
//               // actually handle the error
//            }
//        }
//    }
//
// Becomes this, which can be composed and reused anywhere:
//
//    var thatCommonErr = And(
//       Not(Error(sql.ErrNoRows)),
//       Or(
//         Error(sql.ErrConflict),
//         Error(sql.ErrTransactionClosed),
//       ),
//    )
//
//    func something() {
//        if err := someJankyFunction(); err != nil {
//            if thatCommonErr.In(err) {
//                // handle the error
//            }
//            // more selectors, or just bail
//        }
//    }
//
// Further, selectors automatically traverse the entire error chain if
// it was ever wrapped, and also include intermediate errors in their
// search. It's rather easy to forget an invocation of errors.Cause()
// on an error in these checks (leading to bugs), so the selector tends
// to be a more robust solution. Full chain search in selectors also
// means that even wrapped sentinel errors like:
//
//     var ErrKindaBad = errors.New("it's kinda bad")
//     var ErrVeryBad = errors.Wrap(ErrKindaBad, "it was kinda bad, now it's very bad")
//
// can be inspected with a trivial query like:
//
//     var isErrVeryBad = Error(ErrVeryBad)
//
// Trying to inspect 'sideways' errors like the above would require manual
// traversal of the error chain or (more commonly) some janky searching
// of the error's string representation. Please note that errsel also
// supports janky string searches via Grep(), if you must use one.
//
// Error classes extend selectors with a method to abstract away entire
// categories of errors by creating annotations within an error's context
// chain. While this could be accomplished with a series of normal selectors
// and carefully crafted error constructors, classes make it easier and
// more flexible.
//
//    var numErr = Anonymous()
//
//    func checkN(n int) error {
//       if n < 0 {
//           return numErr.New("negative")
//       }
//       if n > 5000 {
//           return numErr.New("n is probably too large")
//       }
//       return nil
//    }
package errsel
